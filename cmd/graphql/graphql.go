package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"old-scraper/pkg/ads"
	"old-scraper/pkg/collector"
	"old-scraper/pkg/config"
	"old-scraper/pkg/criteria"
	"old-scraper/pkg/dbmodels"
	"old-scraper/pkg/notifications"
	"old-scraper/pkg/printing"
	"old-scraper/pkg/repo"
	results2 "old-scraper/pkg/results"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/gorm"
)

func main() {
	// getting the path of the main file
	ex, err := os.Executable()
	if err != nil {
		log.Panicln(err)
	}

	dir, err := filepath.Abs(filepath.Dir(ex))
	if err != nil {
		log.Panicln(err)
	}

	// loading the config and checking the current directory for an app.yaml file
	cfg, err := config.GetConfig(dir)
	if err != nil {
		log.Panicln(err)
	}

	ntfyHost := cfg.GetString(config.NTFYHost)
	ntfyPort := cfg.GetString(config.NTFYPort)
	ntfyServiceStatusTopic := cfg.GetString(config.NTFYServiceStatusTopic)

	ntfyURL := fmt.Sprintf("http://%s:%s/%s", ntfyHost, ntfyPort, ntfyServiceStatusTopic)

	criteriaTopicName := "criteria"
	criteriaNoticationURL := fmt.Sprintf("http://%s:%s/%s", ntfyHost, ntfyPort, criteriaTopicName)
	criteriaNotificationService := notifications.NewNotificationService(criteriaNoticationURL)

	http.Post(ntfyURL, "text/plain",
		strings.NewReader("Service Old autovit STARTED 😀"))

	done := make(chan bool, 1)

	signalsChannel := make(chan os.Signal, 1)
	signal.Notify(signalsChannel, syscall.SIGINT, syscall.SIGTERM)
	log.Println("start waiting for signal")
	_, cancel := context.WithCancel(context.Background())

	scRepo := criteria.NewSearchCriteriaRepo(cfg)
	autovitRepo := repo.NewAutovitRepository(cfg)

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/start", start(scRepo, autovitRepo, criteriaNotificationService, cfg)).Methods("GET")
	r.HandleFunc("/startcriteria/{id}", startCriteria(scRepo, autovitRepo, criteriaNotificationService, cfg)).Methods("GET")
	r.HandleFunc("/results", results(autovitRepo)).Methods("GET")
	r.HandleFunc("/sold/{date}", sold(autovitRepo)).Methods("GET")
	r.HandleFunc("/new/{date}", newCars(autovitRepo)).Methods("GET")

	port := cfg.GetString(config.HTTPPort)

	go func() {
		log.Println(fmt.Sprintf("Listening on port %s", port))
		err = http.ListenAndServe(fmt.Sprintf(":%s", port), r)
		if err != nil {
			panic(err)
		}
	}()

	go func() {
		log.Println("Waiting for signal")
		sig := <-signalsChannel
		log.Println("Got signal:", sig)
		log.Println("Terminating...")
		http.Post(ntfyURL, "text/plain",
			strings.NewReader("Service Old autovit STOPPED 😡"))
		cancel()
		done <- true
	}()

	<-done
}

func newCars(repo *repo.AutovitRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := getDate(w, r)
		w.Write([]byte(getNewCars(date, repo)))
	}
}

func sold(repo *repo.AutovitRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		date := getDate(w, r)
		w.Write([]byte(getSoldCars(date, repo)))
	}
}

func results(autovitRepo *repo.AutovitRepository) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(results2.GetPriceEvolution(autovitRepo)))
	}
}

func getNewCarsToday(repository *repo.AutovitRepository) string {
	today := time.Now().Format("2006-01-02")
	return getNewCars(today, repository)
}

func getSoldCarsToday(repository *repo.AutovitRepository) string {
	today := time.Now().Format("2006-01-02")
	return getSoldCars(today, repository)
}

func getNewCars(day string, repository *repo.AutovitRepository) string {
	cars := repository.GetInactiveAdsInDay(day)
	result := ""
	for _, car := range cars {
		result = result + printing.PrintCar(fmt.Sprintf("NEW CAR!!! "), car)
	}
	return result
}

func getSoldCars(day string, repository *repo.AutovitRepository) string {
	cars := repository.GetNewAdsInDay(day)
	result := ""
	for _, car := range cars {

		dateString := car.FirstSeen
		log.Println(fmt.Sprintf("dateString: %+v", dateString))
		firstDayOfAd, err := time.Parse("2006-01-02T15:04:05Z", dateString)
		log.Println(fmt.Sprintf("firstDayOfAd: %+v", firstDayOfAd))

		if err != nil {
			panic(err)
		}
		lastDayOfAd, err := time.Parse("2006-01-02T15:04:05Z", *car.LastSeen)
		if err != nil {
			panic(err)
		}

		//difference := lastDayOfAd.Sub(firstDayOfAd)
		difference := lastDayOfAd.Sub(firstDayOfAd)
		diffStr := fmt.Sprintf("Days on autovit: %d", int64(difference.Hours()/24))

		result = result + printing.PrintCar(fmt.Sprintf("NEW !!! - %s - %s", day, diffStr), car)
	}
	return result
}

func start(criteriaRepo *criteria.SearchCriteriaRepo, autovitRepo *repo.AutovitRepository, criteriaNotificationService *notifications.NotificationsService, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		criteriaNotificationService.PushTextNotification("Started getting data for criterias")
		criterias := criteriaRepo.GetCriterias()

		for _, criteria := range *criterias {
			if criteria.Model == "gle_classe" {
				criteria.Model = "gle"
			}
			if criteria.Model == "e_classe" {
				criteria.Model = "e"
			}
			log.Println(fmt.Sprintf("-------Criteria : %s %s ", criteria.Brand, criteria.Model))
			totalCars, err := getCriteriaResults(criteria, autovitRepo, criteriaNotificationService, cfg)
			if err != nil {
				continue
			}
			criteriaEndMessage := fmt.Sprintf("Done with criteria: %s %s found ads: %d", criteria.Brand, criteria.Model, totalCars)
			criteriaNotificationService.PushSuccessNotification(criteriaEndMessage)
			// write criteria results to db

		}

		criteriaNotificationService.PushTextNotification("Done getting data for all criterias")

		w.Write([]byte("Started..."))
	}
}

func startCriteria(criteriaRepo *criteria.SearchCriteriaRepo, autovitRepo *repo.AutovitRepository, criteriaNotificationService *notifications.NotificationsService, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := getID(w, r)
		criteria := criteriaRepo.GetCriteria(uint(id))

		criteriaNotificationService.PushTextNotification(fmt.Sprintf("Started getting data for criteriaId: %d, %s, %s", id, criteria.Brand, criteria.Model))

		carsMap := make(map[string][]dbmodels.Car)
		totalCars, err := getCriteriaResults(*criteria, autovitRepo, criteriaNotificationService, cfg)
		if err != nil {
			criteriaNotificationService.PushErrNotification(err.Error())
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		criteriaNotificationService.PushTextNotification(fmt.Sprintf("Done getting data for criteriaId: %d, %s, %s total found: %d", id, criteria.Brand, criteria.Model, totalCars))

		res, err := json.Marshal(&carsMap)
		if err != nil {
			panic(err)
		}
		w.Write(res)
	}
}

type PaginationInfo struct {
	pageSize      int
	currentOffset int
	totalCount    int
}

func getPaginationInfoFromResponse(res *collector.GraphQLResponse) PaginationInfo {
	pageSize := res.Data.AdvertSearch.PageInfo.PageSize
	currentOffset := res.Data.AdvertSearch.PageInfo.CurrentOffset
	totalCount := res.Data.AdvertSearch.TotalCount
	pi := PaginationInfo{
		pageSize:      pageSize,
		currentOffset: currentOffset,
		totalCount:    totalCount,
	}
	return pi
}

func getCriteriaResults(sc criteria.SearchCriteria, repo *repo.AutovitRepository, notificationService *notifications.NotificationsService, cfg config.Config) (int, error) {
	paginationInfo := PaginationInfo{
		pageSize:      0,
		currentOffset: 0,
		totalCount:    0,
	}
	var foundCars []ads.Ad
	page := 1
	firstResult := getCriteriaPageResults(sc, page)
	foundCars = append(foundCars, firstResult.GetCarAds()...)
	paginationInfo = getPaginationInfoFromResponse(firstResult)
	log.Println(paginationInfo)

	for paginationInfo.pageSize+paginationInfo.currentOffset < paginationInfo.totalCount {
		page++
		res := getCriteriaPageResults(sc, page)
		foundCars = append(foundCars, res.GetCarAds()...)
		paginationInfo = getPaginationInfoFromResponse(res)
	}
	repo.UpsertCarAds(foundCars)
	err := repo.DisableActiveAds(foundCars, sc)
	if err != nil {
		criteriaErrMessage := fmt.Sprintf("Failed to get criteria: %s %s", sc.Brand, sc.Model)
		retryURL := fmt.Sprintf("http://%s/startcriteria/%d", cfg.GetString(config.AppURL), sc.ID)
		notificationService.PushErrRetryNotification(criteriaErrMessage, retryURL)
		return 0, err
	}

	return len(foundCars), nil
}

func getCriteriaPageResults(sc criteria.SearchCriteria, page int) *collector.GraphQLResponse {
	var response collector.GraphQLResponse
	gqlURL := collector.CreateGraphqlURL(sc, page)

	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, gqlURL, nil)

	if err != nil {
		fmt.Println(err)
		return nil
	}

	res, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	defer res.Body.Close()

	err = json.NewDecoder(res.Body).Decode(&response)
	if err != nil {
		panic(err)
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		fmt.Println(err)
		return nil
	}
	fmt.Println(string(body))

	return &response
}

func upsertExistingCarAds(foundCarAds []ads.Ad, db *gorm.DB, today string) map[string][]dbmodels.Car {
	carsMap := make(map[string][]dbmodels.Car)
	for _, carAd := range foundCarAds {
		carAd.ProcessedAt = today
		var existingCarAd dbmodels.Car
		log.Println("finding autovitid ", carAd.Autovit_id)
		errNotFound := db.Where("autovit_id", carAd.Autovit_id).Preload("Prices").Preload("Seller").Last(&existingCarAd).Error

		if errors.Is(errNotFound, gorm.ErrRecordNotFound) {
			dbCarAd := carAd.ToCar(today, today, nil)

			var seller *dbmodels.Seller
			if carAd.DealerAvurl != nil {
				seller = &dbmodels.Seller{
					Aurl: carAd.DealerAvurl,
					Name: carAd.DealerName,
				}
				sellerNotFoundErr := db.Where(dbmodels.Seller{Aurl: carAd.DealerAvurl}).First(&seller)
				if sellerNotFoundErr.Error != nil {
					if carAd.DealerAvurl != nil && carAd.DealerName != nil {
						db.Create(&seller)
					}
				}
			}

			dbCarAd.Seller = seller

			db.Create(&dbCarAd)
			price := dbmodels.Price{
				AdID:  dbCarAd.ID,
				Price: carAd.Price,
				Date:  today,
			}
			db.Table("prices").Create(&price)

			var car dbmodels.Car
			db.Preload("Prices").Preload("Seller").First(&car, dbCarAd.ID)

			carsMap["added"] = append(carsMap["added"], car)
		} else {
			if existingCarAd.ID == 0 {

				var seller *dbmodels.Seller
				if carAd.DealerAvurl != nil {
					seller = &dbmodels.Seller{
						Aurl: carAd.DealerAvurl,
						Name: carAd.DealerName,
					}
					sellerNotFoundErr := db.Where(dbmodels.Seller{Aurl: carAd.DealerAvurl}).First(&seller)
					if sellerNotFoundErr.Error != nil {
						if carAd.DealerAvurl != nil && carAd.DealerName != nil {
							db.Create(&seller)
						}
					}
				}

				// carAd not in db so insert
				dbCarAd := carAd.ToCar(today, today, nil)
				dbCarAd.Seller = seller
				tx := db.Create(&dbCarAd)

				price := dbmodels.Price{
					AdID:  dbCarAd.ID,
					Price: carAd.Price,
					Date:  today,
				}
				db.Table("prices").Create(&price)
				if tx.Error != nil {
					log.Println(tx.Error)
				}
				var car dbmodels.Car
				db.Preload("Prices").Preload("Seller").First(&car, dbCarAd.ID)
				carsMap["added"] = append(carsMap["added"], car)
			} else {

				var seller *dbmodels.Seller
				if carAd.DealerAvurl != nil {
					seller = &dbmodels.Seller{
						Aurl: carAd.DealerAvurl,
						Name: carAd.DealerName,
					}
					sellerNotFoundErr := db.Where(dbmodels.Seller{Aurl: carAd.DealerAvurl}).First(&seller)
					if sellerNotFoundErr.Error != nil {
						if carAd.DealerAvurl != nil && carAd.DealerName != nil {
							db.Create(&seller)
						}
					}
				} else {
					seller = nil
					existingCarAd.SellerID = 0
				}

				existingCarAd.ProcessedAt = today
				existingCarAd.Fuel = carAd.Fuel
				existingCarAd.Active = true
				existingCarAd.LastSeen = nil
				existingCarAd.Seller = seller
				db.Save(&existingCarAd)
				if len(existingCarAd.Prices) == 0 {
					// no prices so might be the first price
					db.Table("prices").Create(&dbmodels.Price{
						AdID:  existingCarAd.ID,
						Price: carAd.Price,
						Date:  today,
					})
				} else {
					if existingCarAd.Prices[len(existingCarAd.Prices)-1].Price != carAd.Price {
						//for _, existingPrice := range existingCarAd.Prices {
						//	log.Printf("Date: %s - %d Price: \n", existingPrice.Date, existingPrice.Price)
						//}
						db.Table("prices").Create(&dbmodels.Price{
							AdID:  existingCarAd.ID,
							Price: carAd.Price,
							Date:  today,
						})
						var car dbmodels.Car
						db.Preload("Prices").Preload("Seller").First(&car, existingCarAd.ID)
					}
				}
			}
		}
	}
	return carsMap
}

func disableExistingCarAds(foundCarAds []ads.Ad, db *gorm.DB, criteria criteria.SearchCriteria) error {
	var existingAds []dbmodels.Car
	db.Table("cars").
		Where(&dbmodels.Car{Active: true, Brand: criteria.Brand, CarModel: criteria.Model, Fuel: criteria.Fuel}).
		Where("year >= ?", criteria.YearFrom).Where("km <= ?", criteria.MileageTo).Find(&existingAds)

	for _, existingCarAd := range existingAds {
		found := false
		for _, foundCarAd := range foundCarAds {
			if foundCarAd.Autovit_id == existingCarAd.Autovit_id {
				found = true
				continue
			}
		}
		if !found {
			if existingCarAd.Active {
				existingCarAd.Active = false
				today := time.Now().Format("2006-01-02")
				existingCarAd.LastSeen = &today
				db.Table("cars").Save(existingCarAd)
			}
		}
	}
	return nil
}

func getID(w http.ResponseWriter, r *http.Request) uint64 {
	vars := mux.Vars(r)
	idStr, ok := vars["id"]
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
	}

	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		http.Error(w, "must be integer", http.StatusBadRequest)
	}

	return id
}

func getDate(w http.ResponseWriter, r *http.Request) string {
	vars := mux.Vars(r)
	dateStr, ok := vars["date"]
	if !ok {
		http.Error(w, "invalid id", http.StatusBadRequest)
	}
	if dateStr == "today" {
		return getToday()
	}
	isDate := regexp.MustCompile(`^\d{4}\-(0?[1-9]|1[012])\-(0?[1-9]|[12][0-9]|3[01])$`).MatchString(dateStr)
	if !isDate {
		http.Error(w, "invalid id", http.StatusBadRequest)
	}
	return dateStr
}

func getToday() string {
	return time.Now().Format("2006-01-02")
}

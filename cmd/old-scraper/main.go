package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"old-scraper/pkg/ads"
	"old-scraper/pkg/collector"
	"old-scraper/pkg/config"
	"old-scraper/pkg/criteria"
	"old-scraper/pkg/dbmodels"
	"old-scraper/pkg/notifications"
	"old-scraper/pkg/pagination"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type Response struct {
	body []byte
}

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

	dbUserName := cfg.GetString(config.DBUsername)
	dbPass := cfg.GetString(config.DBPass)
	dbHost := cfg.GetString(config.DBHost)
	dbName := cfg.GetString(config.DBName)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUserName, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	scRepo := criteria.NewSearchCriteriaRepo(db)

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/start", start(scRepo, criteriaNotificationService, cfg)).Methods("GET")
	r.HandleFunc("/startcriteria/{id}", startCriteria(scRepo, criteriaNotificationService, cfg)).Methods("GET")

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

func startCriteria(criteriaRepo *criteria.SearchCriteriaRepo, criteriaNotificationService *notifications.NotificationsService, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		id := getID(w, r)
		criteria := criteriaRepo.GetCriteria(uint(id))

		criteriaNotificationService.PushTextNotification(fmt.Sprintf("Started getting data for criteriaId: %d, %s, %s", id, criteria.Brand, criteria.Model))

		carsMap := make(map[string][]dbmodels.Car)
		cars := getDataForCriteria(*criteria, criteriaRepo.DB, criteriaNotificationService, cfg)
		carsMap["added"] = append(carsMap["added"], cars["added"]...)
		criteriaNotificationService.PushTextNotification(fmt.Sprintf("Done getting data for criteriaId: %d, %s, %s", id, criteria.Brand, criteria.Model))

		res, err := json.Marshal(&carsMap)
		if err != nil {
			panic(err)
		}
		w.Write(res)
	}
}

func start(criteriaRepo *criteria.SearchCriteriaRepo, criteriaNotificationService *notifications.NotificationsService, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		criteriaNotificationService.PushTextNotification("Started getting data for criterias")
		criterias := criteriaRepo.GetCriterias()
		carsMap := make(map[string][]dbmodels.Car)
		for _, criteria := range *criterias {
			cars := getDataForCriteria(criteria, criteriaRepo.DB, criteriaNotificationService, cfg)
			carsMap["added"] = append(carsMap["added"], cars["added"]...)
		}
		criteriaNotificationService.PushTextNotification("Done getting data for criterias")

		res, err := json.Marshal(&carsMap)
		if err != nil {
			panic(err)
		}
		w.Write(res)
	}
}

func getDataForCriteria(criteria criteria.SearchCriteria, db *gorm.DB, notificationService *notifications.NotificationsService, cfg config.Config) map[string][]dbmodels.Car {

	today := time.Now().Format("2006-01-02")

	foundCarAds := getCarAds(criteria)
	log.Println(fmt.Sprintf("Found %d cars for: %s %s ", len(foundCarAds), criteria.Brand, criteria.Model))

	carsMap := upsertExistingCarAds(foundCarAds, db, today)

	err := disableExistingCarAds(foundCarAds, db, criteria)
	if err != nil {
		criteriaErrMessage := fmt.Sprintf("Failed to get criteria: %s %s", criteria.Brand, criteria.Model)
		retryURL := fmt.Sprintf("http://%s/startcriteria/%d", cfg.GetString(config.AppURL), criteria.ID)
		notificationService.PushErrRetryNotification(criteriaErrMessage, retryURL)
	}
	criteriaEndMessage := fmt.Sprintf("Done with criteria: %s %s", criteria.Brand, criteria.Model)
	notificationService.PushSuccessNotification(criteriaEndMessage)
	return carsMap

}

func upsertExistingCarAds(foundCarAds []ads.Ad, db *gorm.DB, today string) map[string][]dbmodels.Car {
	carsMap := make(map[string][]dbmodels.Car)
	for _, carAd := range foundCarAds {
		carAd.ProcessedAt = today
		var existingCarAd dbmodels.Car

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

func getCarAds(criteria criteria.SearchCriteria) []ads.Ad {
	paginatedCriteria := pagination.Pagination{
		SearchCriteria: criteria,
		PageNumber:     nil,
	}
	pn := 1

	var foundCarAds []ads.Ad
	isLastPage := false

	for !isLastPage {
		if pn > 1 {
			paginatedCriteria.PageNumber = &pn
		}
		carsOnPage, isLast, hasSeveralPages := collector.GetCars(paginatedCriteria)
		foundCarAds = append(foundCarAds, carsOnPage...)
		isLastPage = isLast
		pn++
		if !hasSeveralPages {
			isLastPage = true
		}
	}
	return foundCarAds
}

func disableExistingCarAds(foundCarAds []ads.Ad, db *gorm.DB, criteria criteria.SearchCriteria) error {
	var existingAds []dbmodels.Car
	db.Table("cars").
		Where(&dbmodels.Car{Active: true, Brand: criteria.Brand, CarModel: criteria.Model, Fuel: criteria.Fuel}).
		Where("year >= ?", criteria.YearFrom).Where("km <= ?", criteria.MileageTo).Find(&existingAds)

	if len(existingAds)-len(foundCarAds) > 5 {
		log.Println("something went wrong... ")
		log.Println(fmt.Sprintf("Rescrape for %s %s ", criteria.Brand, criteria.Model))
		log.Println(fmt.Sprintf("Found cars : %d -> Existing cars:  %d ", len(foundCarAds), len(existingAds)))
		return errors.New("Failed for criteria ")
	}

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

package main

import (
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
	"old-scraper/pkg/pagination"
	"os"
	"path/filepath"
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

	dbUserName := cfg.GetString(config.DBUsername)
	dbPass := cfg.GetString(config.DBPass)
	dbHost := cfg.GetString(config.DBHost)
	dbName := cfg.GetString(config.DBName)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUserName, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})

	scRepo := criteria.NewSearchCriteriaRepo(db)

	r := mux.NewRouter().StrictSlash(true)
	r.HandleFunc("/start", start(scRepo)).Methods("GET")

	port := cfg.GetString(config.HTTPPort)
	log.Println(fmt.Sprintf("Listening on port %s", port))
	err = http.ListenAndServe(fmt.Sprintf(":%s", port), r)
	if err != nil {
		panic(err)
	}

}

func start(criteriaRepo *criteria.SearchCriteriaRepo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		criterias := criteriaRepo.GetCriterias()
		carsMap := make(map[string][]dbmodels.Car)
		for _, criteria := range *criterias {
			cars := crawl(criteria, criteriaRepo.DB)
			carsMap["added"] = append(carsMap["added"], cars["added"]...)
		}
		res, err := json.Marshal(&carsMap)
		if err != nil {
			panic(err)
		}
		w.Write(res)
	}
}

func crawl(criteria criteria.SearchCriteria, db *gorm.DB) map[string][]dbmodels.Car {

	today := time.Now().Format("2006-01-02")

	foundCarAds := getCarAds(criteria)
	log.Println(fmt.Sprintf("Found %d cars for: %s %s ", len(foundCarAds), criteria.Brand, criteria.Model))

	carsMap := upsertExistingCarAds(foundCarAds, db, today)

	disableExistingCarAds(foundCarAds, db, today, criteria)

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

func disableExistingCarAds(foundCarAds []ads.Ad, db *gorm.DB, today string, criteria criteria.SearchCriteria) {
	var existingAds []dbmodels.Car
	db.Table("cars").
		Where(&dbmodels.Car{Active: true, Brand: criteria.Brand, CarModel: criteria.Model, Fuel: criteria.Fuel}).
		Where("year >= ?", criteria.YearFrom).Where("km <= ?", criteria.MileageTo).Find(&existingAds)

	if len(existingAds)-len(foundCarAds) > 5 {
		log.Println("something went wrong... ")
		log.Println(fmt.Sprintf("Rescrape for %s %s ", criteria.Brand, criteria.Model))
		log.Println(fmt.Sprintf("Found cars : %d -> Existing cars:  %d ", len(foundCarAds), len(existingAds)))
		return
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
}

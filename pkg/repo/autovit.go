package repo

import (
	"fmt"
	"old-scraper/pkg/ads"
	"old-scraper/pkg/config"
	"old-scraper/pkg/criteria"
	"old-scraper/pkg/dbmodels"
	"time"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type AutovitRepository struct {
	db *gorm.DB
}

func NewAutovitRepository(cfg config.Config) *AutovitRepository {
	dbUserName := cfg.GetString(config.DBUsername)
	dbPass := cfg.GetString(config.DBPass)
	dbHost := cfg.GetString(config.DBHost)
	dbName := cfg.GetString(config.DBName)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUserName, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	return &AutovitRepository{db: db}
}

func (a AutovitRepository) UpsertCarAds(ads []ads.Ad) *[]dbmodels.Car {
	today := time.Now().Format("2006-01-02")
	for _, ad := range ads {
		var existingCarAd dbmodels.Car
		var seller *dbmodels.Seller
		if ad.DealerAvurl != nil {
			//upsert seller
			a.db.Where(dbmodels.Seller{Aurl: ad.DealerAvurl}).First(&seller)
			if seller.ID == 0 {
				seller.Aurl = ad.DealerAvurl
				seller.Name = ad.DealerName
				a.db.Create(&seller)
			}
		} else {
			seller = nil
		}

		if ad.DealerAvurl != nil {
			seller = &dbmodels.Seller{
				Aurl: ad.DealerAvurl,
				Name: ad.DealerName,
			}

		} else {
			seller = nil
			existingCarAd.SellerID = 0
		}

		// get ad by autovitID
		a.db.Where("autovit_id", ad.Autovit_id).Preload("Prices").Preload("Seller").Last(&existingCarAd)
		fsStr, err := time.Parse("2006-01-02T15:04:05Z07:00", existingCarAd.FirstSeen)
		if err != nil {
			panic(err)
		}
		for _, price := range existingCarAd.Prices {
			dateStr, err := time.Parse("2006-01-02T15:04:05Z07:00", price.Date)
			if err != nil {
				panic(err)
			}
			price.Date = dateStr.Format("2006-01-02")
		}
		existingCarAd.FirstSeen = fsStr.Format("2006-01-02")

		// we found the ad, so update

		if existingCarAd.ID > 0 {

			existingCarAd.ProcessedAt = today
			existingCarAd.Fuel = ad.Fuel
			existingCarAd.Active = true
			existingCarAd.LastSeen = nil
			existingCarAd.Seller = seller
			a.db.Save(&existingCarAd)
			if len(existingCarAd.Prices) == 0 {
				// no prices so might be the first price
				a.db.Table("prices").Create(&dbmodels.Price{
					AdID:  existingCarAd.ID,
					Price: ad.Price,
					Date:  today,
				})
			} else {
				if existingCarAd.Prices[len(existingCarAd.Prices)-1].Price != ad.Price {
					a.db.Table("prices").Create(&dbmodels.Price{
						AdID:  existingCarAd.ID,
						Price: ad.Price,
						Date:  today,
					})
				}
			}

		} else {
			// this is a new ad so insert

			dbCarAd := ad.ToCar(today, today, nil)

			dbCarAd.Seller = seller

			a.db.Create(&dbCarAd)
			price := dbmodels.Price{
				AdID:  dbCarAd.ID,
				Price: ad.Price,
				Date:  today,
			}
			a.db.Table("prices").Create(&price)
		}

	}

	return nil
}

func (a AutovitRepository) DisableActiveAds(ads []ads.Ad, criteria criteria.SearchCriteria) error {
	var existingAds []dbmodels.Car
	a.db.Table("cars").
		Where(&dbmodels.Car{Active: true, Brand: criteria.Brand, CarModel: criteria.Model, Fuel: criteria.Fuel}).
		Where("year >= ?", criteria.YearFrom).Where("km <= ?", criteria.MileageTo).Find(&existingAds)

	for _, existingCarAd := range existingAds {
		found := false
		for _, foundCarAd := range ads {
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
				a.db.Table("cars").Save(existingCarAd)
			}
		}
	}
	return nil
}

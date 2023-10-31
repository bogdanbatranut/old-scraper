package repo

import (
	"old-scraper/pkg/ads"
	"old-scraper/pkg/dbmodels"
)

type IRepository interface {
	UpsertCarAds(ads []ads.Ad) *[]dbmodels.Car
	DisableActiveAds(ads []ads.Ad) error
}

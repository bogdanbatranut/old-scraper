package ads

import "old-scraper/pkg/dbmodels"

type Ad struct {
	Brand       string
	Model       string
	Year        int
	Km          int
	Fuel        string
	Price       int
	ProcessedAt string
	Autovit_id  int
	Active      bool
	Ad_url      string
	SellerType  string
	DealerName  *string
	DealerAvurl *string
}

func (ad Ad) ToCar(firstSeen string, processedAt string, lastSeen *string) dbmodels.Car {
	seller := dbmodels.Seller{
		Url:  nil,
		Aurl: ad.DealerAvurl,
		Name: ad.DealerName,
	}
	return dbmodels.Car{
		Brand:       ad.Brand,
		CarModel:    ad.Model,
		Year:        ad.Year,
		Km:          ad.Km,
		FirstSeen:   firstSeen,
		ProcessedAt: processedAt,
		LastSeen:    lastSeen,
		Autovit_id:  ad.Autovit_id,
		Active:      true,
		Ad_url:      ad.Ad_url,
		Fuel:        ad.Fuel,
		Seller:      &seller,
	}
}

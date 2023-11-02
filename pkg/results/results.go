package results

import (
	"fmt"
	"old-scraper/pkg/printing"
	"old-scraper/pkg/repo"
	"sort"
)

func GetPriceEvolution(repo *repo.AutovitRepository) string {
	var result string
	cars := repo.GetActiveAds()

	priceChangeDetected := false

	var priceHistory []printing.PriceDiffHistory

	for _, car := range *cars {
		initialPrice := car.Prices[0].Price
		lastPrice := car.Prices[len(car.Prices)-1].Price
		totalDiff := lastPrice - initialPrice

		if totalDiff >= 0 {
			continue
		}

		ph := printing.PriceDiffHistory{
			PriceDiff:   totalDiff,
			OlderPrices: nil,
			Car:         "",
			AutovitID:   0,
			AdURL:       "",
		}

		for _, price := range car.Prices {
			historyLine := fmt.Sprintf("%s %d", price.Date[0:10], price.Price)
			ph.OlderPrices = append(ph.OlderPrices, historyLine)
		}

		ph.Car = fmt.Sprintf("%s %s %d %dkm", car.Brand, car.CarModel, car.Year, car.Km)
		ph.AutovitID = car.Autovit_id
		ph.AdURL = car.Ad_url
		privat := "privat"
		ph.Seller = &privat
		if car.Seller != nil && car.Seller.Name != nil {
			ph.Seller = car.Seller.Name
		}
		//evolutionSymbolStr := " = "

		priceHistory = append(priceHistory, ph)
		priceChangeDetected = true
	}

	if !priceChangeDetected {
		result = fmt.Sprintf("=========================================================\n" + "no price changes detected...\n" + "=========================================================")
	} else {
		sort.Slice(priceHistory, func(i, j int) bool {
			return priceHistory[i].PriceDiff < priceHistory[j].PriceDiff
		})

		for _, ph := range priceHistory {
			result = result + printing.PrintCarPriceHistory(ph)
		}
		return result
	}
	return result
}

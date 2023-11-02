package printing

import (
	"fmt"
	"old-scraper/pkg/dbmodels"
)

type PriceDiffHistory struct {
	PriceDiff   int
	OlderPrices []string
	Car         string
	AutovitID   int
	AdURL       string
	Seller      *string
}

func PrintAd(car dbmodels.Car, title string) {

	var priceHistoryStr []string

	for _, price := range car.Prices {
		historyLine := fmt.Sprintf("%s %d", price.Date[0:10], price.Price)
		priceHistoryStr = append(priceHistoryStr, historyLine)
	}

	delimiter := " =========================================================================================================================\n"
	softDelimiter := "|-------------------------------------------------------------------------------------------------------------------------|\n"
	autovitIDStr := fmt.Sprintf("| %-120d|\n", car.Autovit_id)
	carDescriptionStr := fmt.Sprintf("%s", fmt.Sprintf("%s %s %d %dkm", car.Brand, car.CarModel, car.Year, car.Km))
	carStr := fmt.Sprintf("| %-120s|\n", carDescriptionStr)
	addUrl := fmt.Sprintf("| %-120s|\n", car.Ad_url)

	priceHistory := ""
	for _, price := range priceHistoryStr {
		formattedHistoryLine := fmt.Sprintf("| %-120s|\n", price)
		priceHistory += formattedHistoryLine
	}
	seller := fmt.Sprintf("| Seller: %-113s|\n", "privat")
	if car.Seller != nil && car.Seller.Name != nil {
		seller = fmt.Sprintf("| Seller:  %-113s|\n", *car.Seller.Name)
	}

	titleFStr := fmt.Sprintf("| %-120s|\n", title)

	fmt.Println(fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s", delimiter, titleFStr, delimiter, priceHistory, softDelimiter, carStr, softDelimiter, autovitIDStr, addUrl, seller, delimiter))

}

func PrintCar(title string, car dbmodels.Car) {
	initialPrice := car.Prices[0].Price
	lastPrice := car.Prices[len(car.Prices)-1].Price
	totalDiff := lastPrice - initialPrice

	ph := PriceDiffHistory{
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
	//evolutionSymbolStr := " = "
	colorValue := 0
	if ph.PriceDiff > 0 {
		colorValue = 31
	}
	if ph.PriceDiff < 0 {
		colorValue = 32
	}

	priceEvolution := fmt.Sprintf("\x1b[%dm%d\x1b[0m", colorValue, ph.PriceDiff)
	priceFormatStr := fmt.Sprintf("Price difference : %s", priceEvolution)

	priceStr := fmt.Sprintf("| %-127s|\n", priceFormatStr)
	delimiter := " =========================================================================================================================\n"
	softDelimiter := "|-------------------------------------------------------------------------------------------------------------------------|\n"
	autovitIDStr := fmt.Sprintf("| %-120d|\n", ph.AutovitID)
	carDescriptionStr := fmt.Sprintf("%s", ph.Car)
	carStr := fmt.Sprintf("| %-120s|\n", carDescriptionStr)
	addUrl := fmt.Sprintf("| %-120s|\n", ph.AdURL)
	seller := fmt.Sprintf("| Seller: %-113s|\n", "privat")
	if car.Seller != nil {
		seller = fmt.Sprintf("| Seller:  %-113s|\n", *car.Seller.Name)
	}

	priceHistory := ""
	for _, price := range ph.OlderPrices {
		formattedHistoryLine := fmt.Sprintf("| %-120s|\n", price)
		priceHistory += formattedHistoryLine
	}

	titleFStr := fmt.Sprintf("| %-120s|\n", title)

	fmt.Println(fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s%s%s", delimiter, titleFStr, delimiter, priceStr, priceHistory, softDelimiter, carStr, softDelimiter, autovitIDStr, addUrl, seller, delimiter))

}

func PrintCarPriceHistory(priceDiffInfo PriceDiffHistory) string {
	//colorValue := 0
	//if priceDiffInfo.PriceDiff > 0 {
	//	colorValue = 31
	//}
	//if priceDiffInfo.PriceDiff < 0 {
	//	colorValue = 32
	//}

	//priceEvolution := fmt.Sprintf("\x1b[%dm%d\x1b[0m", colorValue, priceDiffInfo.PriceDiff)
	//priceEvolution := fmt.Sprintf("\x1b[%dm%d\033[0m", colorValue, priceDiffInfo.PriceDiff)
	priceFormatStr := fmt.Sprintf("Price : %d", priceDiffInfo.PriceDiff)

	priceStr := fmt.Sprintf("| %-129s|\n", priceFormatStr)
	delimiter := " =========================================================================================================================\n"
	softDelimiter := "|-------------------------------------------------------------------------------------------------------------------------|\n"
	autovitIDStr := fmt.Sprintf("| %-120d|\n", priceDiffInfo.AutovitID)
	carDescriptionStr := fmt.Sprintf("%s", priceDiffInfo.Car)
	carStr := fmt.Sprintf("| %-120s|\n", carDescriptionStr)
	addUrl := fmt.Sprintf("| %-120s|\n", priceDiffInfo.AdURL)

	priceHistory := ""
	for _, price := range priceDiffInfo.OlderPrices {
		formattedHistoryLine := fmt.Sprintf("| %-120s|\n", price)
		priceHistory += formattedHistoryLine
	}

	seller := fmt.Sprintf("| Seller: %-113s|\n", "privat")
	if priceDiffInfo.Seller != nil {
		seller = fmt.Sprintf("| Seller:  %-113s|\n", *priceDiffInfo.Seller)
	}

	return fmt.Sprintf("%s%s%s%s%s%s%s%s%s%s", delimiter, priceStr, priceHistory, softDelimiter, carStr, softDelimiter, autovitIDStr, addUrl, seller, delimiter)
}

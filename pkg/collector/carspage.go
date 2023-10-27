package collector

import (
	"fmt"
	"log"
	"old-scraper/pkg/ads"
	"old-scraper/pkg/pagination"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gocolly/colly"
)

func GetCars(sc pagination.Pagination) ([]ads.Ad, bool, bool) {

	collector := colly.NewCollector()

	var carAds []ads.Ad
	isLastPage := false
	hasSeveralPages := false

	collector.OnHTML("ul.pagination-list", func(e *colly.HTMLElement) {
		selection := e.DOM.Find("li[data-testid=\"pagination-step-forwards\"]").HasClass("pagination-item__disabled")
		isLastPage = selection
		hasSeveralPages = true
	})

	collector.OnHTML("article", func(e *colly.HTMLElement) {
		carAd := ads.Ad{
			Brand: sc.SearchCriteria.Brand,
			Model: sc.SearchCriteria.Model,
		}

		// dealer vvvvvv

		ol := e.DOM.Find("article").ChildrenFiltered("ol")
		dealerName, _ := ol.Last().Find("img").Attr("alt")
		link, _ := ol.Last().Find("a").Attr("href")

		privat := "privat"
		carAd.DealerName = &privat
		if link != "" && !strings.HasPrefix(link, "https://www.autovit.ro/") {
			dn := "Profesionist"
			carAd.DealerName = &dn
			carAd.DealerAvurl = &link
		}

		if dealerName != "" {
			carAd.DealerName = &dealerName
		}
		// dealer ˆˆˆˆˆˆˆˆˆˆ

		autovitIDStr := e.Attr("data-id")
		if len(autovitIDStr) > 0 {
			autovitID, err := strconv.Atoi(autovitIDStr)
			if err != nil {
				panic(err)
			}
			carAd.Autovit_id = autovitID
		}

		kmVal := e.DOM.Find("dd[data-parameter=\"mileage\"]").Text()

		kmVal = strings.ReplaceAll(kmVal, " ", "")
		if len(kmVal) > 0 {
			kmVal = strings.ReplaceAll(kmVal, "km", "")
			km, err := strconv.Atoi(kmVal)

			if err != nil {
				panic(err)
			}
			carAd.Km = km
		}

		// year
		yearVal := e.DOM.Find("dd[data-parameter=\"year\"]").Text()
		yearVal = strings.ReplaceAll(yearVal, " ", "")
		if len(yearVal) > 0 {
			year, err := strconv.Atoi(yearVal)
			if err != nil {
				panic(err)
			}
			carAd.Year = year

		}

		priceInTag := e.DOM.Find("div > h3").Text()
		priceStr := strings.ReplaceAll(priceInTag, " ", "")
		if len(priceStr) > 0 {
			price, err := strconv.Atoi(priceStr)
			if err != nil {
				panic(err)
			}
			carAd.Price = price
		}

		adURLTag := e.DOM.Find("div > h1 > a[href]")
		adURL, exists := adURLTag.Attr("href")

		if exists {
			carAd.Ad_url = adURL
		}
		carAd.Active = true
		if carAd.Year != 0 {
			today := time.Now().Format("2006-01-02")
			carAd.ProcessedAt = today
			carAd.Fuel = sc.SearchCriteria.Fuel
			carAds = append(carAds, carAd)

		}
	})
	var bodyArr []byte
	collector.OnResponse(func(response *colly.Response) {
		log.Println("On Response: ", sc.ToURL())
		log.Println(fmt.Sprintf("Status code : %d", response.StatusCode))
		bodyArr = response.Body
	})

	//collector.Visit("https://www.autovit.ro/autoturisme/volvo/xc-60/de-la-2019?search%5Bfilter_enum_fuel_type%5D=diesel&search%5Bfilter_float_mileage%3Ato%5D=125000")
	collector.Visit(sc.ToURL())
	collector.Wait()

	log.Println("Found : ", len(carAds))
	if len(carAds) == 0 {
		err := os.WriteFile("body.html", bodyArr, 0644)
		if err != nil {
			panic(err)
		}
	}
	return carAds, isLastPage, hasSeveralPages

}

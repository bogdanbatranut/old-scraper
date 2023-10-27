package pagination

import (
	"fmt"
	"old-scraper/pkg/criteria"
)

type Pagination struct {
	SearchCriteria criteria.SearchCriteria
	PageNumber     *int
}

func (sc Pagination) ToURL() string {
	pageURL := fmt.Sprintf("https://www.autovit.ro/autoturisme/%s/%s/de-la-%d?search%%5Bfilter_enum_fuel_type%%5D=%s&search%%5Bfilter_float_mileage%%3Ato%%5D=%d", sc.SearchCriteria.Brand, sc.SearchCriteria.Model, sc.SearchCriteria.YearFrom, sc.SearchCriteria.Fuel, sc.SearchCriteria.MileageTo)
	if sc.PageNumber != nil {
		pageURL = fmt.Sprintf("%s&page=%d", pageURL, *sc.PageNumber)
	}
	return pageURL
}

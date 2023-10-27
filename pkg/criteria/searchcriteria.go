package criteria

import (
	"gorm.io/gorm"
)

type SearchCriteria struct {
	Brand        string
	Model        string
	YearFrom     int `gorm:"column:yearfrom"`
	Fuel         string
	MileageTo    int
	AllowProcess bool
}

type SearchCriteriaRepo struct {
	DB *gorm.DB
}

func NewSearchCriteriaRepo(db *gorm.DB) *SearchCriteriaRepo {
	return &SearchCriteriaRepo{
		DB: db,
	}
}

func (repo SearchCriteriaRepo) GetCriterias() *[]SearchCriteria {
	var sc *[]SearchCriteria

	result := repo.DB.Where("allow_process", true).Find(&sc)
	if result.Error != nil {
		panic(result.Error)
	}

	return sc
}

type Tabler interface {
	TableName() string
}

func (sc SearchCriteria) TableName() string {
	return "searchcriterias"
}

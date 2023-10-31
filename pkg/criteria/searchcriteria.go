package criteria

import (
	"fmt"
	"old-scraper/pkg/config"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

type SearchCriteria struct {
	ID           uint
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

func NewSearchCriteriaRepo(cfg config.Config) *SearchCriteriaRepo {
	dbUserName := cfg.GetString(config.DBUsername)
	dbPass := cfg.GetString(config.DBPass)
	dbHost := cfg.GetString(config.DBHost)
	dbName := cfg.GetString(config.DBName)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUserName, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

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

func (repo SearchCriteriaRepo) GetCriteria(id uint) *SearchCriteria {
	var sc *SearchCriteria

	result := repo.DB.First(&sc, id)
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

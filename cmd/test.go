package main

import (
	"errors"
	"fmt"
	"log"
	"old-scraper/pkg/config"
	"old-scraper/pkg/dbmodels"
	"os"
	"path/filepath"

	"gorm.io/driver/mysql"
	"gorm.io/gorm"
)

func main() {
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

	//ntfyHost := cfg.GetString(config.NTFYHost)
	//ntfyPort := cfg.GetString(config.NTFYPort)
	//ntfyServiceStatusTopic := cfg.GetString(config.NTFYServiceStatusTopic)
	//
	//ntfyURL := fmt.Sprintf("http://%s:%s/%s", ntfyHost, ntfyPort, ntfyServiceStatusTopic)

	//criteriaTopicName := "criteria"
	//criteriaNoticationURL := fmt.Sprintf("http://%s:%s/%s", ntfyHost, ntfyPort, criteriaTopicName)
	//criteriaNotificationService := notifications.NewNotificationService(criteriaNoticationURL)

	//http.Post(ntfyURL, "text/plain",
	//	strings.NewReader("Service Old autovit STARTED 😀"))

	//done := make(chan bool, 1)

	//signalsChannel := make(chan os.Signal, 1)
	//signal.Notify(signalsChannel, syscall.SIGINT, syscall.SIGTERM)
	//log.Println("start waiting for signal")
	//_, cancel := context.WithCancel(context.Background())

	dbUserName := cfg.GetString(config.DBUsername)
	dbPass := cfg.GetString(config.DBPass)
	dbHost := cfg.GetString(config.DBHost)
	dbName := cfg.GetString(config.DBName)

	dsn := fmt.Sprintf("%s:%s@tcp(%s:3306)/%s?charset=utf8mb4&parseTime=True&loc=Local", dbUserName, dbPass, dbHost, dbName)
	db, err := gorm.Open(mysql.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	var existingCarAd dbmodels.Car
	err = db.Where("autovit_id", 11111).Preload("Prices").Preload("Seller").Last(&existingCarAd).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("Not found")
		}
	}

	db.Where("autovit_id", 11111).Preload("Prices").Preload("Seller").Last(&existingCarAd)

	//log.Println(existingCarAd)
}

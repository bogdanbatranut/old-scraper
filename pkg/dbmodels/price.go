package dbmodels

type Price struct {
	ID    uint `gorm:"primaryKey"`
	AdID  uint
	Price int
	Date  string `gorm:"column:date"`
	Car   *Car   `gorm:"foreignKey:ad_id"`
}

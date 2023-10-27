package dbmodels

type Seller struct {
	ID   uint `gorm:"primaryKey"`
	Name *string
	Url  *string
	Aurl *string
	Cars []Car `gorm:"foreignKey:seller_id"`
}

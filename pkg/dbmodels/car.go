package dbmodels

type Car struct {
	ID          uint `gorm:"primaryKey"`
	Brand       string
	CarModel    string `gorm:"column:model"`
	Year        int
	Km          int
	Fuel        string
	Prices      []Price `gorm:"foreignKey:ad_id"`
	FirstSeen   string
	ProcessedAt string
	LastSeen    *string
	Autovit_id  int
	Active      bool
	Ad_url      string
	SellerID    uint
	Seller      *Seller `gorm:"foreignKey:seller_id"`
}

func (c Car) ToNotification() {

}

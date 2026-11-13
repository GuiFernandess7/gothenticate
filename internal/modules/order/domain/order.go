package domain

import "time"

type Order struct {
	ID          int64
	UserID      int64
	Status      string
	TotalAmount float64
	Weight      float64
	Zip         string
	Carrier     string
	CreatedAt   time.Time
}

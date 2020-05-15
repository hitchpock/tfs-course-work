package robot

import (
	"encoding/json"
	"errors"
)

var (
	ErrNotFound  = errors.New("not found object")
	ErrInvalidID = errors.New("invalid id")
)

type Storage interface {
	Create(robot *Robot) error
	FindByID(id int) (*Robot, error)
	FindActivatedByUserID(userID int) ([]Robot, error)
	FindActivatedByTicker(ticker string) ([]Robot, error)
	FindActivated() ([]Robot, error)
	FindActivatedByTickerUserID(ticker string, id int) ([]Robot, error)
	Filter(ticker, userID string) ([]Robot, error)
	FavouriteRobot(parentRobotID, userID int) error
	ActivateRobot(robotID int) error
	DeactivateRobot(robotID int) error
	FindToTrading() ([]Robot, error)
	Trade(robot *Robot) error
	SoftDelete(id int) error
}

// Robot структура торгового робота
type Robot struct {
	RobotID       int      `json:"robot_id"`
	OwnerUserID   int      `json:"owner_user_id"`
	ParentRobotID int      `json:"parent_robot_id"`
	IsFavourite   bool     `json:"is_favourite"` //nolint:misspell
	IsActive      bool     `json:"is_active"`
	Ticker        string   `json:"ticker"`
	BuyPrice      float64  `json:"buy_price"`
	SellPrice     float64  `json:"sell_price"`
	PlanStart     NullTime `json:"plan_start"`
	PlanEnd       NullTime `json:"plan_end"`
	PlanYield     float64  `json:"plan_yield"`
	FactYield     float64  `json:"fact_yield"`
	DealsCount    int      `json:"deals_count"`
	ActivatedAt   NullTime `json:"-"`
	DeactivatedAt NullTime `json:"-"`
	CreatedAt     NullTime `json:"-"`
	DeletedAt     NullTime `json:"-"`
	IsBuying      bool     `json:"-"`
}

func (r *Robot) MarshalJSON() ([]byte, error) {
	type Alias Robot

	return json.Marshal(&struct {
		*Alias
		ActivatedAt   NullTime `json:"activated_at,omitempty"`
		DeactivatedAt NullTime `json:"deactivated_at,omitempty"`
		CreatedAt     NullTime `json:"created_at"`
		DeletedAt     NullTime `json:"deleted_at,omitempty"`
	}{
		Alias:         (*Alias)(r),
		ActivatedAt:   r.ActivatedAt,
		DeactivatedAt: r.DeactivatedAt,
		CreatedAt:     r.CreatedAt,
		DeletedAt:     r.DeletedAt,
	})
}

func (r *Robot) Buy(buyPrice float64) {
	r.FactYield -= buyPrice
	r.IsBuying = false
}

func (r *Robot) Sell(sellPrice float64) {
	r.FactYield += sellPrice
	r.DealsCount++
	r.IsBuying = true
}

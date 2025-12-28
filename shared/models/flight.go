package models

import "time"

// Flight represents an available flight
type Flight struct {
	ID             string    `json:"id"`
	FlightNumber   string    `json:"flightNumber"`
	Origin         string    `json:"origin"`
	Destination    string    `json:"destination"`
	DepartureTime  time.Time `json:"departureTime"`
	ArrivalTime    time.Time `json:"arrivalTime"`
	TotalSeats     int       `json:"totalSeats"`
	AvailableSeats int       `json:"availableSeats"`
	PricePerSeat   float64   `json:"pricePerSeat"`
}

// Seat represents a seat on a flight
type Seat struct {
	ID       string     `json:"id"`
	FlightID string     `json:"flightId"`
	Row      int        `json:"row"`
	Column   string     `json:"column"`
	Class    SeatClass  `json:"class"`
	Status   SeatStatus `json:"status"`
	Price    float64    `json:"price"`
}

type SeatClass string

const (
	SeatClassEconomy  SeatClass = "economy"
	SeatClassBusiness SeatClass = "business"
	SeatClassFirst    SeatClass = "first"
)

type SeatStatus string

const (
	SeatStatusAvailable SeatStatus = "available"
	SeatStatusHeld      SeatStatus = "held"
	SeatStatusBooked    SeatStatus = "booked"
)


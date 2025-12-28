module github.com/cx-tal-miterani/flight-booking-system/temporal-worker

go 1.21

require (
	github.com/cx-tal-miterani/flight-booking-system/shared v0.0.0
	github.com/stretchr/testify v1.8.4
	go.temporal.io/sdk v1.26.1
)

replace github.com/cx-tal-miterani/flight-booking-system/shared => ../shared


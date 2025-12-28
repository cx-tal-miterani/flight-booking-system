module github.com/cx-tal-miterani/flight-booking-system/api-server

go 1.21

require (
	github.com/cx-tal-miterani/flight-booking-system/shared v0.0.0
	github.com/go-chi/chi/v5 v5.0.12
	github.com/go-chi/cors v1.2.1
	github.com/google/uuid v1.6.0
	github.com/stretchr/testify v1.8.4
	go.temporal.io/sdk v1.26.1
)

replace github.com/cx-tal-miterani/flight-booking-system/shared => ../shared


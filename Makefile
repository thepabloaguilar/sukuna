.PHONY: create-migration
create-migration:
	migrate create -ext sql -dir gateways/postgres/migrations -seq init_schema

.PHONY: migrate
migrate:
	migrate -path gateways/postgres/migration -database "pgx://sukuna:sukuna@localhost:5432/sukuna?sslmode=disable" -verbose up

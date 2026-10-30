.PHONY: seed sqlc server swagger docker-up docker-down docker-build docker-seed

seed:
	go run ./cmd/seeder

sqlc:
	sqlc generate

server:
	go run ./cmd/api

swagger:
	swag init -g cmd/api/main.go

docker-build:
	docker compose build

docker-up:
	docker compose up -d

docker-down:
	docker compose down

docker-seed:
	docker compose run --rm seeder

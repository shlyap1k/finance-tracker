include .env

.PHONY: help docker-up docker-down migrate-up migrate-down gen-sqlc run

help:
	@echo "Доступные команды:"
	@echo "  docker-up      - Поднять БД в Docker"
	@echo "  docker-down    - Остановить и удалить контейнеры БД"
	@echo "  migrate-up     - Накатить миграции локально"
	@echo "  migrate-down   - Откатить последнюю миграцию"
	@echo "  gen-sqlc       - Сгенерировать Go-код из SQL (требует установленный sqlc)"
	@echo "  run            - Запустить приложение"

docker-up:
	docker-compose up -d

docker-down:
	docker-compose down -v

migrate-up:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" up

migrate-down:
	migrate -path migrations -database "postgres://$(DB_USER):$(DB_PASSWORD)@$(DB_HOST):$(DB_PORT)/$(DB_NAME)?sslmode=$(DB_SSLMODE)" down 1

gen-sqlc:
	sqlc generate

run:
	go run cmd/api/main.go
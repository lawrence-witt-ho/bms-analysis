-include .env
export

errors:
	go run cmd/errors/main.go
.PHONY: logs

alerts:
	go run cmd/alerts/main.go
.PHONY: alerts

types:
	go run cmd/types/main.go
.PHONY: types

dev:
	make types && wails dev
.PHONY: dev

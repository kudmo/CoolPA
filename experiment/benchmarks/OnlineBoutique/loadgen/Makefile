# =====================================================
# Locust Load Generator
# =====================================================

.PHONY: help
help:
	@echo "Available commands:"
	@echo "  make build               - Build locust image"
	@echo "  make up                  - Start load generator (with Web UI)"
	@echo "  make up-headless         - Start load generator (headless mode)"
	@echo "  make down                - Stop load generator"
	@echo "  make logs                - View logs"
	@echo "  make status              - Check status"
	@echo "  make set-rate RATE=200   - Smoothly change RATE"
	@echo "  make set-users USERS=20  - Smoothly change USERS"
	@echo "  make set-config          - Update both USERS and RATE"
	@echo "  make show-config         - Show current configuration"
	@echo "  make webui               - Open Locust Web UI"
	@echo "  make test                - Run quick test (30 seconds)"
	@echo "  make benchmark           - Run benchmark with different rates"

CONFIG_DIR ?= ./config
CONFIG_FILE ?= $(CONFIG_DIR)/config.env

$(CONFIG_DIR):
	mkdir -p $(CONFIG_DIR)
	mkdir -p ./reports

.PHONY: help build up down logs status test

help:
	@echo "Available commands:"
	@echo "  make build    - Build locust image"
	@echo "  make up       - Start load generator (with Web UI)"
	@echo "  make down     - Stop load generator"
	@echo "  make logs     - View logs"
	@echo "  make status   - Check status"

build:
	docker build -t locust-loadgen:latest .

up:
	HEADLESS=false docker compose up -d
	@echo "Load generator started"
	@echo "Web UI: http://localhost:8089"

down:
	docker compose down

logs:
	docker compose logs -f

status:
	docker compose ps
	@echo ""
	@echo "Locust Web UI: http://localhost:8089"

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

# Переменные по умолчанию
USERS ?= 10
RATE ?= 100
FRONTEND_ADDR ?= http://localhost:8080
REFRESH_INTERVAL ?= 5
HEADLESS ?= false
RUN_TIME ?= 60s
CONFIG_DIR ?= ./config
CONFIG_FILE ?= $(CONFIG_DIR)/config.env

$(CONFIG_DIR):
	mkdir -p $(CONFIG_DIR)
	mkdir -p ./reports

# Сборка образа
.PHONY: build
build:
	docker build -t locust-loadgen:latest .

# Запуск с Web UI (можно менять параметры через браузер)
.PHONY: up
up: $(CONFIG_DIR)
	@echo "Starting Locust load generator with USERS=$(USERS), RATE=$(RATE)"
	@echo "USERS=$(USERS)" > $(CONFIG_FILE)
	@echo "RATE=$(RATE)" >> $(CONFIG_FILE)
	@echo "FRONTEND_ADDR=$(FRONTEND_ADDR)" >> $(CONFIG_FILE)
	HEADLESS=false \
	FRONTEND_ADDR=$(FRONTEND_ADDR) \
	USERS=$(USERS) \
	RATE=$(RATE) \
	REFRESH_INTERVAL=$(REFRESH_INTERVAL) \
	docker compose up -d
	@echo "✅ Load generator started"
	@echo "🌐 Web UI: http://localhost:8089"
	@echo "📊 Config file: $(CONFIG_FILE)"

# Запуск в headless режиме (без Web UI)
.PHONY: up-headless
up-headless: $(CONFIG_DIR)
	@echo "Starting Locust in headless mode with USERS=$(USERS), RATE=$(RATE)"
	@echo "USERS=$(USERS)" > $(CONFIG_FILE)
	@echo "RATE=$(RATE)" >> $(CONFIG_FILE)
	HEADLESS=true \
	RUN_TIME=$(RUN_TIME) \
	FRONTEND_ADDR=$(FRONTEND_ADDR) \
	USERS=$(USERS) \
	RATE=$(RATE) \
	docker compose up -d
	@echo "✅ Headless load generator started for $(RUN_TIME)"
	@echo "📊 Reports will be saved to ./reports/"

# Остановка
.PHONY: down
down:
	docker compose down

# Просмотр логов
.PHONY: logs
logs:
	docker compose logs -f

# Статус
.PHONY: status
status:
	docker compose ps
	@echo ""
	@echo "Current configuration:"
	@if [ -f $(CONFIG_FILE) ]; then \
		cat $(CONFIG_FILE); \
	else \
		echo "Config file not found"; \
	fi
	@echo ""
	@echo "Locust Web UI: http://localhost:8089"

# Плавное изменение RATE (через обновление конфига)
.PHONY: set-rate
set-rate:
	@if [ -z "$(RATE)" ]; then \
		echo "Usage: make set-rate RATE=200"; \
		exit 1; \
	fi
	@echo "Smoothly changing RATE to $(RATE)..."
	@if [ -f $(CONFIG_FILE) ]; then \
		if grep -q "^RATE=" $(CONFIG_FILE); then \
			sed -i.bak 's/^RATE=.*/RATE=$(RATE)/' $(CONFIG_FILE); \
		else \
			echo "RATE=$(RATE)" >> $(CONFIG_FILE); \
		fi; \
		echo "✅ RATE updated to $(RATE)"; \
		echo "ℹ️  Locust will apply changes within $(REFRESH_INTERVAL) seconds"; \
		echo "📊 Check Web UI to see the change"; \
	else \
		echo "Config file not found. Run 'make up' first"; \
		exit 1; \
	fi

# Плавное изменение USERS
.PHONY: set-users
set-users:
	@if [ -z "$(USERS)" ]; then \
		echo "Usage: make set-users USERS=20"; \
		exit 1; \
	fi
	@echo "Smoothly changing USERS to $(USERS)..."
	@if [ -f $(CONFIG_FILE) ]; then \
		if grep -q "^USERS=" $(CONFIG_FILE); then \
			sed -i.bak 's/^USERS=.*/USERS=$(USERS)/' $(CONFIG_FILE); \
		else \
			echo "USERS=$(USERS)" >> $(CONFIG_FILE); \
		fi; \
		echo "✅ USERS updated to $(USERS)"; \
		echo "ℹ️  Locust will apply changes within $(REFRESH_INTERVAL) seconds"; \
	else \
		echo "Config file not found. Run 'make up' first"; \
		exit 1; \
	fi

# Обновление обоих параметров
.PHONY: set-config
set-config: $(CONFIG_DIR)
	@echo "Updating configuration: USERS=$(USERS), RATE=$(RATE)"
	@echo "USERS=$(USERS)" > $(CONFIG_FILE)
	@echo "RATE=$(RATE)" >> $(CONFIG_FILE)
	@echo "FRONTEND_ADDR=$(FRONTEND_ADDR)" >> $(CONFIG_FILE)
	@echo "✅ Configuration updated"
	@echo "ℹ️  Locust will apply changes within $(REFRESH_INTERVAL) seconds"

# Показать текущую конфигурацию
.PHONY: show-config
show-config:
	@if [ -f $(CONFIG_FILE) ]; then \
		echo "Current configuration:"; \
		cat $(CONFIG_FILE); \
		echo ""; \
		echo "Active environment in container:"; \
		docker exec locust-loadgen env | grep -E "^(USERS|RATE|FRONTEND_ADDR)=" || true; \
	else \
		echo "No config file found"; \
	fi

# Открыть Web UI
.PHONY: webui
webui:
	@echo "Opening Locust Web UI..."
	@xdg-open http://localhost:8089 2>/dev/null || open http://localhost:8089 2>/dev/null || echo "Open http://localhost:8089 in your browser"

# Быстрый тест
.PHONY: test
test: down
	@echo "Running quick test (30 seconds)..."
	$(MAKE) up-headless USERS=$(USERS) RATE=$(RATE) RUN_TIME=30s
	@sleep 35
	$(MAKE) down
	@echo "✅ Test completed"

# Бенчмарк с разными RPS
.PHONY: benchmark
benchmark: down
	@echo "Starting benchmark..."
	@for rate in 10 50 100 200 500; do \
		echo ""; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
		echo "Testing RATE=$$rate with USERS=10"; \
		echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"; \
		$(MAKE) up-headless USERS=10 RATE=$$rate RUN_TIME=60s; \
		sleep 65; \
		$(MAKE) down; \
		sleep 5; \
	done
	@echo "✅ Benchmark completed"

# Очистка
.PHONY: clean
clean:
	@echo "Cleaning..."
	docker compose down
	rm -rf $(CONFIG_DIR)
	rm -rf ./reports
	@echo "✅ Cleaned"

# Мониторинг
.PHONY: watch
watch:
	@echo "Watching config file for changes..."
	@while true; do \
		if [ -f $(CONFIG_FILE) ]; then \
			clear; \
			echo "=== Locust Load Generator Config ==="; \
			cat $(CONFIG_FILE); \
			echo ""; \
			echo "Last updated: $$(date)"; \
			echo "Web UI: http://localhost:8089"; \
			echo "Press Ctrl+C to exit"; \
		fi; \
		sleep 2; \
	done
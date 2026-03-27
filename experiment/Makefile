# =====================================================
# Microservices Demo Benchmark Pipeline
# KinD + Istio + Prometheus + Chaos Mesh
# =====================================================

SHELL := /bin/bash

CLUSTER_NAME ?= ms-demo
KCTX ?= kind-$(CLUSTER_NAME)

NS ?= microservices-demo
MON_NS ?= monitoring
SCALER_NS ?= autoscale-test
CHAOS_NS ?= chaos-mesh

KUBECTL := kubectl --context=$(KCTX)

BENCH_DIR := ./benchmarks/microservices-demo
BENCH_MANIFESTS_DIR := $(BENCH_DIR)/release

ISTIO_PROFILE ?= demo
ISTIO_NS ?= istio-system
ISTIOCTL ?= istioctl

MONITORING_DIR ?= ./monitoring
SCALER_DIR ?= ./smartautoscaler

METRICS_URL ?= https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

AUTOSCALER_IMAGE ?= smartautoscaler:latest

# Chaos Mesh
CHAOS_MESH_VERSION ?= latest
CHAOS_MESH_HELM_REPO := https://charts.chaos-mesh.org
CHAOS_MESH_CHART := chaos-mesh/chaos-mesh

# =====================================================
# HELP
# =====================================================

.PHONY: help
help:
	@echo ""
	@echo "Cluster:"
	@echo "  cluster-up        create KinD cluster"
	@echo "  cluster-down      delete cluster"
	@echo ""
	@echo "Deploy:"
	@echo "  deploy-monitoring deploy prometheus + kube-state-metrics"
	@echo "  deploy-demo       deploy microservices-demo"
	@echo "  deploy-all        monitoring + demo + scaler"
	@echo "  deploy-chaos      deploy Chaos Mesh"
	@echo ""
	@echo "Istio:"
	@echo "  istio-install"
	@echo "  istio-label-ns"
	@echo ""
	@echo "Chaos Mesh:"
	@echo "  chaos-install     install Chaos Mesh via Helm"
	@echo "  chaos-uninstall   uninstall Chaos Mesh"
	@echo "  chaos-dashboard   port-forward Chaos Dashboard"
	@echo "  chaos-status      show Chaos Mesh components status"
	@echo "  chaos-experiments list all chaos experiments"
	@echo "  chaos-clean       delete all chaos experiments"
	@echo ""
	@echo "Debug:"
	@echo "  pods"
	@echo "  svc"
	@echo "  logs-loadgen"
	@echo ""
	@echo "Port forward:"
	@echo "  prometheus"
	@echo "  frontend"
	@echo ""
	@echo "Масштабирование:"
	@echo "  hpa-scale       - горизонтальное масштабирование (изменение количества реплик)"
	@echo "      Параметры: SERVICE=имя_сервиса (default: test-go-app)"
	@echo "                 REPLICAS=количество (default: 3)"
	@echo "      Пример: make hpa-scale SERVICE=test-go-app-client REPLICAS=5"
	@echo ""
	@echo "  vpa-scale       - вертикальное масштабирование (изменение ресурсов CPU/Memory)"
	@echo "      Параметры: SERVICE=имя_сервиса (default: test-go-app)"
	@echo "                 REQUESTS_CPU=запрос CPU (default: 200m)"
	@echo "                 REQUESTS_MEM=запрос памяти (default: 512Mi)"
	@echo "                 LIMITS_CPU=лимит CPU (default: 500m)"
	@echo "                 LIMITS_MEM=лимит памяти (default: 1Gi)"
	@echo "      Пример: make vpa-scale SERVICE=test-go-app-client REQUESTS_CPU=300m LIMITS_MEM=2Gi"
	@echo ""
	@echo "Composite:"
	@echo "  up"
	@echo "  destroy"



# =====================================================
# SCALER
# =====================================================

.PHONY: scaler-build
scaler-build:
	docker build -t $(AUTOSCALER_IMAGE) /home/mk/Files/Work/CoolPA/smartautoscaler

.PHONY: scaler-kind-load
scaler-kind-load:
	kind load docker-image $(AUTOSCALER_IMAGE) --name $(CLUSTER_NAME); \

.PHONY: deploy-scaler
deploy-scaler: scaler-build scaler-kind-load
	$(KUBECTL) create namespace $(SCALER_NS) || true
	$(KUBECTL) apply -f $(SCALER_DIR)/

.PHONY: scaler-logs
scaler-logs:
	$(KUBECTL) -n $(SCALER_NS) logs -l app=smartautoscaler --tail=200

# =====================================================
# Loadgen Integration (Locust-based)
# =====================================================

# Определяем тип loadgen (можно переопределить)
LOADGEN_TYPE ?= locust  # locust или legacy

# Директории
LOADGEN_DIR := $(BENCH_DIR)/loadgen
LOADGEN_CONFIG_DIR ?= $(LOADGEN_DIR)/config
LOADGEN_MAKE := $(MAKE) -C $(LOADGEN_DIR)

# Параметры нагрузки
USERS ?= 10
RATE ?= 100
FRONTEND_ADDR ?= http://localhost:8080  # Для locust нужен полный URL
REFRESH_INTERVAL ?= 5
HEADLESS ?= false
RUN_TIME ?= 60s

# Проверка существования loadgen директории
.PHONY: check-loadgen
check-loadgen:
	@if [ ! -d "$(LOADGEN_DIR)" ]; then \
		echo "❌ Loadgen directory not found: $(LOADGEN_DIR)"; \
		echo "Please ensure LOADGEN_DIR points to the correct location"; \
		echo "Current BENCH_DIR: $(BENCH_DIR)"; \
		exit 1; \
	fi
	@if [ ! -f "$(LOADGEN_DIR)/Makefile" ]; then \
		echo "❌ Loadgen Makefile not found in $(LOADGEN_DIR)"; \
		echo "Please set up loadgen first:"; \
		echo "  cd $(LOADGEN_DIR) && make setup"; \
		exit 1; \
	fi

# Сборка образа
.PHONY: loadgen-build
loadgen-build: check-loadgen
	@echo "🔨 Building loadgen image ($(LOADGEN_TYPE))..."
	@$(LOADGEN_MAKE) build

# Запуск loadgen
.PHONY: loadgen-up
loadgen-up: check-loadgen
	@echo "🚀 Starting loadgenerator ($(LOADGEN_TYPE)) with USERS=$(USERS), RATE=$(RATE)"
	@mkdir -p $(LOADGEN_CONFIG_DIR)
	@if [ "$(LOADGEN_TYPE)" = "locust" ]; then \
		$(LOADGEN_MAKE) up \
			USERS=$(USERS) \
			RATE=$(RATE) \
			FRONTEND_ADDR=$(FRONTEND_ADDR) \
			REFRESH_INTERVAL=$(REFRESH_INTERVAL) \
			HEADLESS=$(HEADLESS); \
	else \
		$(LOADGEN_MAKE) up \
			USERS=$(USERS) \
			RATE=$(RATE) \
			FRONTEND_ADDR=$(FRONTEND_ADDR) \
			REFRESH_INTERVAL=$(REFRESH_INTERVAL); \
	fi
	@echo "✅ Loadgenerator started"
	@if [ "$(LOADGEN_TYPE)" = "locust" ] && [ "$(HEADLESS)" != "true" ]; then \
		echo "🌐 Locust Web UI: http://localhost:8089"; \
	fi

# Запуск в headless режиме (только для locust)
.PHONY: loadgen-up-headless
loadgen-up-headless: check-loadgen
	@echo "🚀 Starting loadgenerator in headless mode with USERS=$(USERS), RATE=$(RATE), RUN_TIME=$(RUN_TIME)"
	@mkdir -p $(LOADGEN_CONFIG_DIR)
	@$(LOADGEN_MAKE) up-headless \
		USERS=$(USERS) \
		RATE=$(RATE) \
		FRONTEND_ADDR=$(FRONTEND_ADDR) \
		RUN_TIME=$(RUN_TIME) \
		REFRESH_INTERVAL=$(REFRESH_INTERVAL)

# Остановка loadgen
.PHONY: loadgen-down
loadgen-down: check-loadgen
	@echo "🛑 Stopping loadgenerator..."
	@$(LOADGEN_MAKE) down || true
	@rm -rf $(LOADGEN_CONFIG_DIR) 2>/dev/null || true

# Плавное изменение RATE
.PHONY: loadgen-set-rate
loadgen-set-rate: check-loadgen
	@if [ -z "$(RATE)" ]; then \
		echo "❌ Error: RATE is required. Usage: make loadgen-set-rate RATE=200"; \
		exit 1; \
	fi
	@echo "📊 Smoothly changing RATE to $(RATE)..."
	@$(LOADGEN_MAKE) set-rate RATE=$(RATE)

# Плавное изменение USERS
.PHONY: loadgen-set-users
loadgen-set-users: check-loadgen
	@if [ -z "$(USERS)" ]; then \
		echo "❌ Error: USERS is required. Usage: make loadgen-set-users USERS=20"; \
		exit 1; \
	fi
	@echo "👥 Smoothly changing USERS to $(USERS)..."
	@$(LOADGEN_MAKE) set-users USERS=$(USERS)

# Обновление обоих параметров
.PHONY: loadgen-set-config
loadgen-set-config: check-loadgen
	@echo "⚙️  Updating configuration: USERS=$(USERS), RATE=$(RATE)"
	@$(LOADGEN_MAKE) set-config USERS=$(USERS) RATE=$(RATE)

# Просмотр логов
.PHONY: loadgen-logs
loadgen-logs: check-loadgen
	@$(LOADGEN_MAKE) logs

# Статус
.PHONY: loadgen-status
loadgen-status: check-loadgen
	@echo "📊 Loadgenerator status:"
	@$(LOADGEN_MAKE) status || echo "Loadgenerator is not running"

# Показать конфигурацию
.PHONY: loadgen-show-config
loadgen-show-config: check-loadgen
	@$(LOADGEN_MAKE) show-config

# Открыть Web UI (только для locust)
.PHONY: loadgen-webui
loadgen-webui: check-loadgen
	@if [ "$(LOADGEN_TYPE)" = "locust" ]; then \
		$(LOADGEN_MAKE) webui; \
	else \
		echo "Web UI only available for Locust-based loadgen"; \
	fi

# Быстрый тест
.PHONY: loadgen-test
loadgen-test: check-loadgen
	@echo "🧪 Running quick test with USERS=$(USERS), RATE=$(RATE)"
	@$(LOADGEN_MAKE) test USERS=$(USERS) RATE=$(RATE)

# Бенчмарк с разными RPS
.PHONY: loadgen-benchmark
loadgen-benchmark: check-loadgen
	@echo "🏃 Running benchmark..."
	@$(LOADGEN_MAKE) benchmark

# Стресс-тест с разными пользователями
.PHONY: loadgen-stress-test
loadgen-stress-test: check-loadgen
	@echo "💪 Running stress test..."
	@$(LOADGEN_MAKE) stress-test

# Мониторинг конфигурации
.PHONY: loadgen-watch
loadgen-watch: check-loadgen
	@$(LOADGEN_MAKE) watch

# Очистка
.PHONY: loadgen-clean
loadgen-clean: check-loadgen
	@echo "🧹 Cleaning loadgen..."
	@$(LOADGEN_MAKE) clean || true
	@rm -rf $(LOADGEN_CONFIG_DIR)
	@echo "✅ Loadgen cleaned"

# Полный сброс
.PHONY: loadgen-reset
loadgen-reset: loadgen-down loadgen-clean
	@echo "🔄 Loadgen fully reset"

# Рестарт
.PHONY: loadgen-restart
loadgen-restart: loadgen-down loadgen-up
	@echo "🔄 Loadgen restarted"

# Помощь по loadgen
.PHONY: loadgen-help
loadgen-help:
	@echo "=========================================="
	@echo "📚 Loadgen Integration Commands"
	@echo "=========================================="
	@echo ""
	@echo "Configuration:"
	@echo "  LOADGEN_TYPE=$(LOADGEN_TYPE)              - Type: locust or legacy"
	@echo "  USERS=$(USERS)                             - Number of simulated users"
	@echo "  RATE=$(RATE)                               - Requests per second"
	@echo "  FRONTEND_ADDR=$(FRONTEND_ADDR)             - Frontend service address"
	@echo "  REFRESH_INTERVAL=$(REFRESH_INTERVAL)       - Config refresh interval (seconds)"
	@echo "  HEADLESS=$(HEADLESS)                       - Run without Web UI (locust only)"
	@echo "  RUN_TIME=$(RUN_TIME)                       - Test duration in headless mode"
	@echo ""
	@echo "Setup & Build:"
	@echo "  make loadgen-build          - Build loadgen Docker image"
	@echo "  make check-loadgen          - Verify loadgen directory exists"
	@echo ""
	@echo "Control:"
	@echo "  make loadgen-up              - Start load generator"
	@echo "  make loadgen-up-headless     - Start in headless mode (locust only)"
	@echo "  make loadgen-down            - Stop load generator"
	@echo "  make loadgen-restart         - Restart load generator"
	@echo ""
	@echo "Dynamic Control:"
	@echo "  make loadgen-set-rate RATE=200     - Smoothly change RPS"
	@echo "  make loadgen-set-users USERS=20    - Smoothly change users"
	@echo "  make loadgen-set-config            - Update both parameters"
	@echo ""
	@echo "Monitoring:"
	@echo "  make loadgen-logs            - View logs"
	@echo "  make loadgen-status          - Check status"
	@echo "  make loadgen-show-config     - Show current configuration"
	@echo "  make loadgen-watch           - Watch config changes"
	@echo "  make loadgen-webui           - Open Locust Web UI (locust only)"
	@echo ""
	@echo "Testing:"
	@echo "  make loadgen-test            - Quick 30-second test"
	@echo "  make loadgen-benchmark       - Benchmark different RPS values"
	@echo "  make loadgen-stress-test     - Stress test different user counts"
	@echo ""
	@echo "Cleanup:"
	@echo "  make loadgen-clean           - Clean loadgen files"
	@echo "  make loadgen-reset           - Full reset (stop + clean)"
	@echo ""
	@echo "Examples:"
	@echo "  make loadgen-up USERS=20 RATE=150"
	@echo "  make loadgen-set-rate RATE=500"
	@echo "  make loadgen-test USERS=10 RATE=50"
	@echo "  make loadgen-up-headless RUN_TIME=300s"
	@echo "=========================================="

# Комбинированные цели для разных сценариев тестирования
.PHONY: test-ramp-up
test-ramp-up: loadgen-up
	@echo "📈 Running ramp-up test..."
	@sleep 10
	@$(MAKE) loadgen-set-rate RATE=50
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=100
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=200
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=300
	@sleep 30
	@echo "✅ Ramp-up test completed"

.PHONY: test-ramp-down
test-ramp-down: loadgen-up
	@echo "📉 Running ramp-down test..."
	@sleep 10
	@$(MAKE) loadgen-set-rate RATE=200
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=100
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=50
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=10
	@sleep 30
	@echo "✅ Ramp-down test completed"

.PHONY: test-spike
test-spike: loadgen-up
	@echo "⚡ Running spike test..."
	@sleep 10
	@$(MAKE) loadgen-set-rate RATE=100
	@sleep 30
	@$(MAKE) loadgen-set-rate RATE=500
	@sleep 10
	@$(MAKE) loadgen-set-rate RATE=100
	@sleep 30
	@echo "✅ Spike test completed"

export USERS
export RATE
export FRONTEND_ADDR
export REFRESH_INTERVAL
export HEADLESS
export RUN_TIME
export LOADGEN_TYPE

# =====================================================
# CLUSTER
# =====================================================

.PHONY: cluster-up
cluster-up:
	kind create cluster --name $(CLUSTER_NAME)
	$(KUBECTL) cluster-info

.PHONY: cluster-down
cluster-down:
	kind delete cluster --name $(CLUSTER_NAME)

# =====================================================
# ISTIO
# =====================================================

.PHONY: istio-install
istio-install:
	$(ISTIOCTL) install \
	--set profile=$(ISTIO_PROFILE) \
	--set values.pilot.resources.requests.cpu=100m \
	--set values.pilot.resources.requests.memory=128Mi \
	--set values.pilot.resources.limits.cpu=500m \
	--set values.pilot.resources.limits.memory=512Mi \
	--set values.global.proxy.resources.requests.cpu=100m \
	--set values.global.proxy.resources.requests.memory=128Mi \
	--set values.global.proxy.resources.limits.cpu=500m \
	--set values.global.proxy.resources.limits.memory=256Mi \
	-y

.PHONY: istio-label-ns
istio-label-ns:
	$(KUBECTL) label namespace $(NS) istio.io/rev=default --overwrite
	$(KUBECTL) annotate namespace $(NS) sidecar.istio.io/proxyCPU=100m --overwrite
	$(KUBECTL) annotate namespace $(NS) sidecar.istio.io/proxyCPULimit=500m --overwrite
	$(KUBECTL) annotate namespace $(NS) sidecar.istio.io/proxyMemory=128Mi --overwrite
	$(KUBECTL) annotate namespace $(NS) sidecar.istio.io/proxyMemoryLimit=256Mi --overwrite
	$(KUBECTL) rollout restart deployment -n $(NS)

.PHONY: istio-apply-manifests
istio-apply-manifests:
	$(KUBECTL) apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/v1.1.0/standard-install.yaml
	$(KUBECTL) apply -f $(BENCH_MANIFESTS_DIR)/istio-classic-gateway.yaml
	$(KUBECTL) wait --for=condition=ready pod -l app=frontend -n $(NS) --timeout=60s

# =====================================================
# CHAOS MESH
# =====================================================

.PHONY: chaos-install
chaos-install:
	@echo "Installing Chaos Mesh..."
	helm repo add chaos-mesh $(CHAOS_MESH_HELM_REPO) || true
	helm repo update
	$(KUBECTL) create namespace $(CHAOS_NS) || true
	helm install chaos-mesh $(CHAOS_MESH_CHART) \
		--namespace $(CHAOS_NS) \
		--set dashboard.create=true \
		--set chaosDaemon.runtime=containerd \
		--set chaosDaemon.socketPath=/run/containerd/containerd.sock \
		--set dashboard.securityMode=false
	@echo "Waiting for Chaos Mesh pods to be ready..."
	$(KUBECTL) -n $(CHAOS_NS) wait --for=condition=ready pod --all --timeout=120s
	@echo "Chaos Mesh installed successfully!"

.PHONY: chaos-uninstall
chaos-uninstall:
	@echo "Uninstalling Chaos Mesh..."
	helm uninstall chaos-mesh -n $(CHAOS_NS) || true
	$(KUBECTL) delete ns $(CHAOS_NS) || true
	@echo "Chaos Mesh uninstalled."

.PHONY: chaos-dashboard
chaos-dashboard:
	@echo "Port-forwarding Chaos Dashboard to http://localhost:2333"
	$(KUBECTL) -n $(CHAOS_NS) port-forward svc/chaos-dashboard 2333:2333

.PHONY: chaos-status
chaos-status:
	@echo "Chaos Mesh components:"
	$(KUBECTL) -n $(CHAOS_NS) get pods
	@echo ""
	@echo "Chaos experiments:"
	$(KUBECTL) get podchaos,networkchaos,stresschaos,iochaos -A 2>/dev/null || echo "No experiments found"

.PHONY: chaos-experiments
chaos-experiments:
	@echo "=== PodChaos ==="
	$(KUBECTL) get podchaos -A 2>/dev/null || echo "No PodChaos"
	@echo ""
	@echo "=== NetworkChaos ==="
	$(KUBECTL) get networkchaos -A 2>/dev/null || echo "No NetworkChaos"
	@echo ""
	@echo "=== StressChaos ==="
	$(KUBECTL) get stresschaos -A 2>/dev/null || echo "No StressChaos"
	@echo ""
	@echo "=== IoChaos ==="
	$(KUBECTL) get iochaos -A 2>/dev/null || echo "No IoChaos"

.PHONY: chaos-clean
chaos-clean:
	@echo "Deleting all chaos experiments..."
	$(KUBECTL) delete podchaos --all -A 2>/dev/null || true
	$(KUBECTL) delete networkchaos --all -A 2>/dev/null || true
	$(KUBECTL) delete stresschaos --all -A 2>/dev/null || true
	$(KUBECTL) delete iochaos --all -A 2>/dev/null || true
	@echo "All experiments deleted."

# =====================================================
# MONITORING
# =====================================================

.PHONY: deploy-monitoring
deploy-monitoring:
	$(KUBECTL) create namespace $(MON_NS) || true
	$(KUBECTL) apply -f $(MONITORING_DIR)/prometheus
	$(KUBECTL) apply -f $(MONITORING_DIR)/kube-state-metrics
	$(KUBECTL) apply -k $(MONITORING_DIR)/metrics-server

# =====================================================
# DEMO
# =====================================================

.PHONY: deploy-demo
deploy-demo:
# 	$(KUBECTL) create namespace $(NS) || true
	$(KUBECTL) apply -f $(BENCH_MANIFESTS_DIR)/namespace.yaml
	$(KUBECTL) apply -n $(NS) -f $(BENCH_MANIFESTS_DIR)/kubernetes-manifests.yaml

# =====================================================
# PORT FORWARD
# =====================================================

.PHONY: prometheus
prometheus:
	$(KUBECTL) -n $(MON_NS) port-forward svc/prometheus 9090:9090

.PHONY: frontend
frontend:
	$(KUBECTL) port-forward -n $(ISTIO_NS) svc/istio-ingressgateway 8080:80

# =====================================================
# DEBUG
# =====================================================

.PHONY: pods
pods:
	$(KUBECTL) -n $(NS) get pods -o wide

.PHONY: svc
svc:
	$(KUBECTL) -n $(NS) get svc

.PHONY: logs-loadgen
logs-loadgen:
	$(KUBECTL) -n $(NS) logs -l app=loadgenerator --tail=200

# =====================================================
# SCALING
# =====================================================

SERVICE ?= test-go-app
REPLICAS ?= 3
REQUESTS_CPU ?= 200m
REQUESTS_MEM ?= 512Mi
LIMITS_CPU ?= 500m
LIMITS_MEM ?= 1Gi

.PHONY: hpa-scale
hpa-scale:
	$(KUBECTL) -n $(NS) scale deploy/$(SERVICE) --replicas=$(REPLICAS)
	$(KUBECTL) -n $(NS) rollout status deploy/$(SERVICE)

.PHONY: vpa-scale
vpa-scale:
	$(KUBECTL) -n $(NS) set resources deploy/$(SERVICE) \
		--requests=cpu=$(REQUESTS_CPU),memory=$(REQUESTS_MEM) \
		--limits=cpu=$(LIMITS_CPU),memory=$(LIMITS_MEM)
	$(KUBECTL) -n $(NS) rollout status deploy/$(SERVICE)

# =====================================================
# EXPERIMENT
# =====================================================

.PHONY: experiment
experiment:
	python experiment_runner/experiment_runner.py \
		--namespace $(NS)

# =====================================================
# COMPOSITE
# =====================================================

.PHONY: deploy-all
deploy-all: deploy-monitoring deploy-demo

.PHONY: up
up: cluster-up istio-install deploy-all istio-label-ns istio-apply-manifests
	@echo ""
	@echo "System ready"
	@echo ""
	@echo "Prometheus:"
	@echo "make prometheus"
	@echo ""
	@echo "Frontend:"
	@echo "make frontend"
	@echo ""
	@echo "Chaos Mesh:"
	@echo "make chaos-install"
	@echo "make chaos-dashboard"

.PHONY: destroy
destroy: cluster-down
	@echo "Cluster deleted"

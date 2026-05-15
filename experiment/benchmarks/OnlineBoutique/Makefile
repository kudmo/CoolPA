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

BENCH_DIR := .
BENCH_MANIFESTS_DIR := $(BENCH_DIR)/release

ISTIO_PROFILE ?= demo
ISTIO_NS ?= istio-system
ISTIOCTL ?= istioctl

MONITORING_DIR ?= $(BENCH_DIR)/monitoring
SCALER_DIR ?= $(BENCH_DIR)/smartautoscaler
SCALER_APP_DIR ?= $(BENCH_DIR)/../../../smartautoscaler

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
	# 	@echo "HPA Management:"
	@echo "  hpa-apply               - apply HPA config from file"
	@echo "  hpa-delete              - delete all HPA in namespace"
	@echo "  hpa-get                 - show all HPA"
	@echo "  hpa-watch               - watch HPA status"
	@echo ""
	@echo "HPA Parameters:"
	@echo "  TARGET_CPU=70           - target CPU utilization percentage"
	@echo "  MIN=1                   - minimum replicas"
	@echo "  MAX=5                   - maximum replicas"
	@echo "  HPA_DIR=./path          - directory for HPA configs"
	@echo "  HPA_CONFIG=./file.yaml  - path to HPA config file"
	@echo ""
	@echo "Composite:"
	@echo "  up"
	@echo "  destroy"



# =====================================================
# SCALER
# =====================================================

.PHONY: scaler-build
scaler-build:
	docker build -t $(AUTOSCALER_IMAGE) $(SCALER_APP_DIR)

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

LOADGEN_DIR := $(BENCH_DIR)/loadgen
LOADGEN_CONFIG_DIR ?= $(LOADGEN_DIR)/config
LOADGEN_MAKE := $(MAKE) -C $(LOADGEN_DIR)

.PHONY: check-loadgen
check-loadgen:
	@if [ ! -d "$(LOADGEN_DIR)" ]; then \
		echo "Loadgen directory not found: $(LOADGEN_DIR)"; \
		echo "Please ensure LOADGEN_DIR points to the correct location"; \
		echo "Current BENCH_DIR: $(BENCH_DIR)"; \
		exit 1; \
	fi
	@if [ ! -f "$(LOADGEN_DIR)/Makefile" ]; then \
		echo "Loadgen Makefile not found in $(LOADGEN_DIR)"; \
		echo "Please set up loadgen first:"; \
		echo "  cd $(LOADGEN_DIR) && make setup"; \
		exit 1; \
	fi

.PHONY: loadgen-build
loadgen-build: check-loadgen
	@echo "🔨 Building loadgen image..."
	@$(LOADGEN_MAKE) build

.PHONY: loadgen-up
loadgen-up: check-loadgen
	@echo "Starting loadgenerator"
	@mkdir -p $(LOADGEN_CONFIG_DIR)
	$(LOADGEN_MAKE) up
	@echo "Loadgenerator started"
	@echo "Locust Web UI: http://localhost:8089"; \

.PHONY: loadgen-down
loadgen-down: check-loadgen
	@echo "Stopping loadgenerator..."
	@$(LOADGEN_MAKE) down || true
	@rm -rf $(LOADGEN_CONFIG_DIR) 2>/dev/null || true

.PHONY: loadgen-logs
loadgen-logs: check-loadgen
	@$(LOADGEN_MAKE) logs

.PHONY: loadgen-status
loadgen-status: check-loadgen
	@echo "Loadgenerator status:"
	@$(LOADGEN_MAKE) status || echo "Loadgenerator is not running"


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
	$(KUBECTL) wait --for=condition=ready pod -l app=frontend -n $(NS) --timeout=300s

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
	$(KUBECTL) -n $(CHAOS_NS) wait --for=condition=ready pod --all --timeout=300s
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
	make istio-label-ns
	$(KUBECTL) apply -n $(NS) -f $(BENCH_MANIFESTS_DIR)/kubernetes-manifests.yaml

# =====================================================
# PORT FORWARD
# =====================================================

.PHONY: prometheus
prometheus:
	$(KUBECTL) -n $(MON_NS) port-forward svc/prometheus 9090:9090 > /dev/null 2>&1 &

.PHONY: frontend
frontend:
	$(KUBECTL) port-forward -n $(ISTIO_NS) svc/istio-ingressgateway 8080:80 > /dev/null 2>&1 &

# =====================================================
# DEBUG
# =====================================================

.PHONY: pods
pods:
	$(KUBECTL) -n $(NS) get pods -o wide

.PHONY: svc
svc:
	$(KUBECTL) -n $(NS) get svc


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
# HPA (Horizontal Pod Autoscaler) Management
# =====================================================

HPA_DIR ?= $(BENCH_DIR)/hpa
HPA_CONFIG ?= $(HPA_DIR)/hpa.yaml
HPA_TARGET_CPU ?= 70

# Список сервисов для HPA
HPA_SERVICES ?= adservice cartservice checkoutservice currencyservice emailservice \
                frontend paymentservice productcatalogservice recommendationservice \
                redis-cart shippingservice

.PHONY: hpa-apply
hpa-apply:
	@echo "Applying HPA configurations..."
	@if [ -f "$(HPA_CONFIG)" ]; then \
		$(KUBECTL) apply -f $(HPA_CONFIG); \
	else \
		echo "HPA config not found at $(HPA_CONFIG)"; \
		echo "Creating default HPA for all services..."; \
		$(MAKE) hpa-create-all; \
	fi
	$(KUBECTL) -n $(NS) get hpa

.PHONY: hpa-delete
hpa-delete:
	@echo "🗑️  Deleting all HPA in namespace $(NS)..."
	$(KUBECTL) -n $(NS) delete hpa --all
	@echo "✅ All HPA deleted"

.PHONY: hpa-get
hpa-get:
	@echo "📊 HPA status in namespace $(NS):"
	$(KUBECTL) -n $(NS) get hpa
	@echo ""
	@echo "📈 Detailed HPA info:"
	$(KUBECTL) -n $(NS) describe hpa

.PHONY: hpa-watch
hpa-watch:
	@echo "👀 Watching HPA status (press Ctrl+C to stop)..."
	$(KUBECTL) -n $(NS) get hpa -w

.PHONY: hpa-metrics
hpa-metrics:
	@echo "📊 Current metrics for pods in $(NS):"
	@echo "=== Pod Metrics ==="
	$(KUBECTL) -n $(NS) top pods
	@echo ""
	@echo "=== Node Metrics ==="
	$(KUBECTL) top nodes

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

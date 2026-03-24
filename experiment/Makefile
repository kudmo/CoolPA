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
BENCH_MANIFESTS_DIR := $(BENCH_DIR)/release/kubernetes-manifests.yaml

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
# Loadgen
# =====================================================

USERS ?= 1
RATE ?= 10

.PHONY: loadgen-setup
loadgen-setup:
	$(KUBECTL) -n $(NS) set env deployment/loadgenerator USERS=$(USERS) RATE=$(RATE)

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
	-y

.PHONY: istio-label-ns
istio-label-ns:
	$(KUBECTL) label namespace $(NS) istio.io/rev=default --overwrite
	$(KUBECTL) rollout restart deployment -n $(NS)

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
	$(KUBECTL) create namespace $(NS) || true
	$(KUBECTL) apply -n $(NS) -f $(BENCH_MANIFESTS_DIR)

# =====================================================
# PORT FORWARD
# =====================================================

.PHONY: prometheus
prometheus:
	$(KUBECTL) -n $(MON_NS) port-forward svc/prometheus 9090:9090

.PHONY: frontend
frontend:
	$(KUBECTL) -n $(NS) port-forward svc/frontend 8080:80

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
up: cluster-up istio-install deploy-all istio-label-ns
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

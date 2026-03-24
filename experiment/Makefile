# =====================================================
# Microservices Demo Benchmark Pipeline
# KinD + Istio + Prometheus
# =====================================================

SHELL := /bin/bash

CLUSTER_NAME ?= ms-demo
KCTX ?= kind-$(CLUSTER_NAME)

NS ?= microservices-demo
MON_NS ?= monitoring
SCALER_NS ?= autoscale-test

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
	@echo ""
	@echo "Istio:"
	@echo "  istio-install"
	@echo "  istio-label-ns"
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

.PHONY: destroy
destroy: cluster-down
	@echo "Cluster deleted"

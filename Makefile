# Makefile для управления KinD-кластером, Prometheus и SmartAutoscaler
# Требуется: docker, kind, kubectl
# Использование: make help

SHELL := /bin/bash

# Параметры
CLUSTER_NAME ?= autoscale-test
KCTX ?= kind-$(CLUSTER_NAME)
NS ?= autoscale-test

TEST_APP_IMAGE ?= test-go-app:latest
AUTOSCALER_IMAGE ?= smartautoscaler:latest

K8S_DIR ?= k8s

KUBECTL := kubectl --context=$(KCTX)

.PHONY: help
help:
	@echo "Основные цели:"
	@echo "  up              - создать кластер, собрать образы и задеплоить всё"
	@echo "  down            - удалить namespace"
	@echo "  destroy         - полный снос (namespace + кластер)"
	@echo ""
	@echo "KinD:"
	@echo "  cluster-up      - создать кластер"
	@echo "  cluster-down    - удалить кластер"
	@echo "  cluster-status  - список кластеров"
	@echo ""
	@echo "Образы:"
	@echo "  build           - собрать Docker-образы"
	@echo "  kind-load       - загрузить образы в KinD"
	@echo ""
	@echo "Deploy:"
	@echo "  deploy          - применить все манифесты"
	@echo "  undeploy        - удалить namespace"
	@echo "  redeploy        - удалить и применить заново"
	@echo ""
	@echo "Port-forward:"
	@echo "  prometheus      - открыть Prometheus UI (localhost:9090)"
	@echo "  app             - открыть test-go-app (localhost:8080)"
	@echo "  autoscaler      - открыть smartautoscaler (localhost:8088)"
	@echo ""
	@echo "Отладка:"
	@echo "  pods            - список pod'ов"
	@echo "  logs-app        - логи test-go-app"
	@echo "  logs-autoscaler - логи smartautoscaler"
	@echo "  logs-prom       - логи Prometheus"
	@echo ""
	@echo "Масштабирование:"
	@echo "  scale-up        - масштабировать test-go-app до 3 реплик"
	@echo "  scale-down      - вернуть 1 реплику"
	@echo ""
	@echo "Istio:"
	@echo "  istio-install         - установить Istio"
	@echo "  istio-uninstall       - удалить Istio"
	@echo "  istio-status          - статус компонентов Istio"
	@echo "  istio-label-ns        - включить sidecar injection для namespace"


# =========================
# Istio
# =========================

.PHONY: istio-install
istio-install:
	@echo "Installing Istio (profile=$(ISTIO_PROFILE))..."
	$(ISTIOCTL) install \
		--set profile=$(ISTIO_PROFILE) \
		--set values.global.proxy.resources.requests.cpu=10m \
		--set values.global.proxy.resources.requests.memory=64Mi \
		-y

.PHONY: istio-uninstall
istio-uninstall:
	@echo "Removing Istio..."
	-$(ISTIOCTL) uninstall --purge -y || true
	-@kubectl delete namespace $(ISTIO_NS) --ignore-not-found

.PHONY: istio-status
istio-status:
	@kubectl -n $(ISTIO_NS) get pods
	@kubectl -n $(ISTIO_NS) get svc

.PHONY: istio-label-ns
istio-label-ns:
	@echo "Enabling sidecar injection in namespace $(NS)..."
	@kubectl label namespace $(NS) istio-injection=enabled --overwrite

# =========================
# КЛАСТЕР
# =========================

.PHONY: cluster-up
cluster-up:
	set -euo pipefail; \
	kind create cluster --name $(CLUSTER_NAME); \
	$(KUBECTL) cluster-info

.PHONY: cluster-down
cluster-down:
	kind delete cluster --name $(CLUSTER_NAME)

.PHONY: cluster-status
cluster-status:
	kind get clusters

# =========================
# ОБРАЗЫ
# =========================

.PHONY: build
build:
	set -euo pipefail; \
	docker build -t $(TEST_APP_IMAGE) ./test-go-app; \
	docker build -t $(AUTOSCALER_IMAGE) ./smartautoscaler

.PHONY: kind-load
kind-load:
	set -euo pipefail; \
	docker pull k8s.gcr.io/kube-state-metrics/kube-state-metrics:v2.8.0; \
	docker pull k8s.gcr.io/metrics-server/metrics-server:v0.6.3; \
	kind load docker-image $(TEST_APP_IMAGE) --name $(CLUSTER_NAME); \
	kind load docker-image $(AUTOSCALER_IMAGE) --name $(CLUSTER_NAME); \
	kind load docker-image k8s.gcr.io/kube-state-metrics/kube-state-metrics:v2.8.0 --name $(CLUSTER_NAME); \
	kind load docker-image k8s.gcr.io/metrics-server/metrics-server:v0.6.3 --name $(CLUSTER_NAME);

# =========================
# DEPLOY
# =========================

.PHONY: deploy
deploy:
	$(KUBECTL) apply -f $(K8S_DIR)/namespace.yaml
	$(KUBECTL) apply -f $(K8S_DIR)/ --recursive

.PHONY: undeploy
undeploy:
	-$(KUBECTL) delete namespace $(NS) --ignore-not-found

.PHONY: redeploy
redeploy:
	$(MAKE) undeploy
	$(MAKE) deploy

# =========================
# PORT FORWARD
# =========================

.PHONY: prometheus
prometheus:
	$(KUBECTL) -n $(NS) port-forward svc/prometheus 9090:9090

.PHONY: app
app:
	$(KUBECTL) -n $(NS) port-forward svc/test-go-app 8080:8080

.PHONY: autoscaler
autoscaler:
	$(KUBECTL) -n $(NS) port-forward svc/smartautoscaler 8088:8088

# =========================
# ОТЛАДКА
# =========================

.PHONY: pods
pods:
	$(KUBECTL) -n $(NS) get pods -o wide

.PHONY: logs-app
logs-app:
	$(KUBECTL) -n $(NS) logs -l app=test-go-app --tail=200

.PHONY: logs-autoscaler
logs-autoscaler:
	$(KUBECTL) -n $(NS) logs -l app=smartautoscaler --tail=200

.PHONY: logs-prom
logs-prom:
	$(KUBECTL) -n $(NS) logs -l app=prometheus --tail=200

# =========================
# МАСШТАБИРОВАНИЕ
# =========================

.PHONY: scale-up
scale-up:
	$(KUBECTL) -n $(NS) scale deploy/test-go-app --replicas=3
	$(KUBECTL) -n $(NS) rollout status deploy/test-go-app

.PHONY: scale-down
scale-down:
	$(KUBECTL) -n $(NS) scale deploy/test-go-app --replicas=1
	$(KUBECTL) -n $(NS) rollout status deploy/test-go-app

# =========================
# КОМПОЗИТНЫЕ ЦЕЛИ
# =========================

.PHONY: up
up: cluster-up build kind-load deploy
	@echo "Cluster + Prometheus + test-go-app + smartautoscaler запущены."

.PHONY: down
down: undeploy
	@echo "Namespace удалён."

.PHONY: destroy
destroy: undeploy cluster-down
	@echo "Полное удаление выполнено."

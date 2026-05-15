
# OnlineBoutique — Benchmark Example

## Overview

- Application: OnlineBoutique (microservices demo)
- Integrations provided: Istio, Prometheus, kube-state-metrics, metrics-server
- Load generator: Locust (see `loadgen/`)
- Autoscaler: `smartautoscaler` (build locally or load into kind)
- Chaos engineering support via Chaos Mesh

## Repository layout

- `hpa/` — HPA examples and SLO-related configs
- `loadgen/` — load generator implementation (Locust), Dockerfile and Makefile
- `monitoring/` — Prometheus, kube-state-metrics and metrics-server manifests
- `release/` — Kubernetes manifests for the demo app and Istio

## Requirements

- `kind` — Kubernetes in Docker (for local cluster)
- `kubectl` — Kubernetes CLI
- `istioctl` — for installing Istio (optional if Istio already present)
- `helm` — for installing Chaos Mesh
- Docker — to build the autoscaler and loadgen images

Ensure your host has enough CPU and RAM to run KinD + Istio + Prometheus.

## Quick start

1. Start the full stack:

```bash
make up
```

2. Build and load the autoscaler image into kind:

```bash
make scaler-build
make scaler-kind-load
make deploy-scaler
```

3. Run the load generator (Locust):

```bash
make loadgen-build
make loadgen-up     # port 8089
```

4. Port-forward Prometheus or the frontend:

```bash
make prometheus     # port 9090
make frontend       # port 8080
```

## Key Make targets

- `cluster-up` / `cluster-down` — create/delete a KinD cluster
- `istio-install` — install Istio
- `deploy-monitoring` — deploy Prometheus + kube-state-metrics + metrics-server
- `deploy-demo` — deploy the OnlineBoutique demo app
- `deploy-scaler` — build and install `smartautoscaler` into the autoscaler namespace
- `loadgen-*` — load generator controls (build, up, down, logs, webui, up-headless)
- `chaos-*` — install/uninstall and manage Chaos Mesh and experiments
- `hpa-*` — apply/remove/view HPA configurations

## Using Chaos Mesh

1. Install Chaos Mesh:

```bash
make chaos-install
```

2. View experiments and status:

```bash
make chaos-status
make chaos-experiments
```

3. Delete all experiments:

```bash
make chaos-clean
```

## HPA

HPA example manifests are in the `hpa/` directory. To apply the default configs:

```bash
make hpa-apply
```

Also available: `hpa-get`, `hpa-delete`, `hpa-watch`.

## Debugging & logs

- View pods: `make pods`
- View services: `make svc`
- Loadgen logs: `make logs-loadgen`
- Autoscaler logs: `make scaler-logs`

## Links and resources

- Original OnlineBoutique: https://github.com/GoogleCloudPlatform/microservices-demo
- Prometheus: https://prometheus.io/
- Istio: https://istio.io/
- Chaos Mesh: https://chaos-mesh.org/



# CoolPA — Smart Autoscaler Research Project

This repository contains components and experiments for researching and benchmarking autoscaling strategies for microservices. The project includes a Go-based autoscaler (`smartautoscaler`), analysis tools, and benchmark scenarios (including a prepared OnlineBoutique demo).

## Contents

- `smartautoscaler/` — main autoscaler implementation (Go), models and config files.
- `experiment/benchmarks/OnlineBoutique/` — prepared benchmark for the Google OnlineBoutique demo (Make targets for KinD, Istio, Prometheus, Chaos Mesh, loadgen).

## Table of Contents
- [CoolPA — Smart Autoscaler Research Project](#coolpa--smart-autoscaler-research-project)
  - [Contents](#contents)
  - [Table of Contents](#table-of-contents)
  - [Overview](#overview)
  - [How it works](#how-it-works)
  - [Quick start](#quick-start)
  - [Building and running `smartautoscaler` locally](#building-and-running-smartautoscaler-locally)
  - [License](#license)

## Overview

The goal of this repository is to provide a complete local environment for experimenting with autoscaling algorithms, monitoring pipelines and chaos scenarios. Typical workflows include:

- Run the OnlineBoutique demo on a local KinD cluster with Istio and Prometheus.
- Generate traffic using the integrated Locust load generator.
- Deploy `smartautoscaler` to control scaling decisions based on monitored metrics and prediction models.
- Introduce chaos experiments with Chaos Mesh to evaluate robustness.

For details on the OnlineBoutique benchmark and how to run it, see the benchmark README:

[experiment/benchmarks/OnlineBoutique/Readme.md](experiment/benchmarks/OnlineBoutique/Readme.md)

## How it works

The autoscaler is designed for Kubernetes microservices where Service Level Objectives (SLO) matter more than raw CPU or memory metrics. It operates as a standalone component inside the cluster and follows a closed control loop composed of three main modules:

- **Collector** – Periodically fetches metrics (Prometheus) and request traces (Istio).
- **Analyzer** – Detects SLO violations and resource over‑provisioning, then identifies bottleneck services using a modified **TopoRank** algorithm that propagates anomalies along the service call graph.
- **Decision** – Uses the bottleneck list and resource usage data to find an optimal scaling strategy. A **genetic algorithm** searches the space of possible actions (horizontal/vertical scaling) while a lightweight **ONNX model** predicts the probability of SLO violation for each candidate action. The final strategy minimises a weighted linear combination of SLO risk and resource cost.

Once the best strategy is found, the autoscaler applies it by patching the corresponding Kubernetes Deployments (replicas, CPU/memory limits). The whole loop repeats every 15–30 seconds, allowing fast reaction to load changes while avoiding unnecessary oscillations.

## Quick start

Prerequisites: `kind`, `kubectl`, `docker`, `istioctl` (optional), `helm`.

1) Start the benchmark stack (creates KinD cluster, installs Istio and monitoring):

```bash
cd experiment/benchmarks/OnlineBoutique
make up
```

2) Build and deploy the autoscaler image (from repository root):

```bash
docker build -t smartautoscaler:local -f smartautoscaler/Dockerfile smartautoscaler
# load into kind (if using kind):
kind load docker-image smartautoscaler:local --name ms-demo
```

3) Deploy the autoscaler inside the cluster (there is a Make target in the benchmark Makefile):

```bash
cd experiment/benchmarks/OnlineBoutique
make deploy-scaler
```

4) Start the load generator and observe metrics (via Prometheus / Grafana if configured):

```bash
make loadgen-build
make loadgen-up
```

## Building and running `smartautoscaler` locally

From the project root you can build the Go binary:

```bash
cd smartautoscaler
go build ./...
# or run directly
go run ./cmd
```

Configuration for the autoscaler lives in `smartautoscaler/config.yaml` and `smartautoscaler/config/`.

There is also a prebuilt model file `smartautoscaler/latency_model.onnx` used by the predictor.

## License

This project is licensed under the Apache License 2.0 — see the [LICENSE](LICENSE) file for details.

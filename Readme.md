
# CoolPA — Smart Autoscaler Research Project

[![Go Version](https://img.shields.io/badge/Go-1.26.6+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Status: Research](https://img.shields.io/badge/Status-Research-orange.svg)]()
[![Release](https://img.shields.io/github/v/release/kudmo/CoolPA?include_prereleases)](https://github.com/kudmo/CoolPA/releases)

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

The autoscaler is designed for microservices where Service Level Objectives (SLOs) are more important than raw CPU or memory usage metrics. It works as a separate component within the cluster and represents a closed control loop consisting of four main modules:

- **MetricsRepository** — provides the required metrics. The project includes an implementation using Prometheus and Istio.
- **Analyzer**: identifies SLO violations using a modified **TopoRank** algorithm, which disseminates information about anomalies across the service call graph. It also identifies services with excess resources.
- **Optimizer**: finds the optimal scaling strategy for the identified problematic services using a **Genetic Algorithm** with an **ONNX model** to predict the likelihood of SLO violations for the selected scaling strategy.
- **Applier**: applies the chosen scaling strategy in the cluster. The project includes an implementation for k8s.

## Quick start
At present, the project has a ready‑made implementation for scaling using k8s, Prometheus, and Istio.
The functionality can be tested using test applications and benchmarks provided in [experiment/benchmarks](experiment/benchmarks).

Prerequisites: `kind`, `kubectl`, `docker`, `istioctl`, `helm`.

Alternative: You can also run the entire experiment in the provided devcontainer, which comes pre‑configured with all necessary tools and dependencies. This is the recommended way for quick experimentation without manual setup.

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

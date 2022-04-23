# multi-tenants

[![Go Report Card](https://goreportcard.com/badge/github.com/k8s-cloud-platform/multi-tenants)](https://goreportcard.com/report/github.com/k8s-cloud-platform/multi-tenants)
[![Releases](https://img.shields.io/github/release/k8s-cloud-platform/multi-tenants/all.svg)](https://github.com/k8s-cloud-platform/multi-tenants/releases)
[![Coverage Status](https://coveralls.io/repos/github/multi-cluster-platform/mcp/badge.svg?branch=main)](https://coveralls.io/github/k8s-cloud-platform/multi-tenants?branch=main)

multi tenants within one cluster



![architecture](docs/images/architecture.png)



## Concepts

- Tenant: tenant k8s cluster
- Project: k8s namespace
- Application: k8s deployment



## Milestone

| release | feature                                                      |
|:--------| :----------------------------------------------------------- |
| v0.0.0  | tenant cluster define, includes certificate、deployment、service、kubeconfig |
| v0.0.1  | define Tenant CRD to manage tenant cluster                   |
| v0.1.0  | tenant syncer, sync spec from tenant cluster to host cluster & status from host cluster to tenant cluster |
| v0.1.1  | aggregate apiserver in host cluster, unify all traffic       |
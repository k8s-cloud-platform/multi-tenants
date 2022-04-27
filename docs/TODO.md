# TODO



## 租户

提供不同的租户隔离方式

- 强隔离：独立apiserver、controller-manager、etcd

- 弱隔离：namespace隔离，共享apiserver、controller-manager、etcd



## 集群

提供不同的集群对接方式

- host：部署到宿主集群
- virtual-kubelet：部署到vk的节点
- OCM：部署到OCM worker集群

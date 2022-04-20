# tenant



## 1. 测试

### 准备工作

#### 部署集群

```shell
kind create cluster --name host
```

#### 准备证书

将 etcd 相关的证书存放到 kube-system/etcd-cert secret 中

- ca.crt: /etc/kubernetes/pki/etcd/ca.crt
- apiserver-etcd-client.crt: /etc/kubernetes/pki/apiserver-etcd-client.crt
- apiserver-etcd-client.key: /etc/kubernetes/pki/apiserver-etcd-client.key

```shell
kubectl create secret generic etcd-cert \
  -n kube-system \
  --from-file=ca.crt=/etc/kubernetes/pki/etcd/ca.crt \
  --from-file=apiserver-etcd-client.crt=/etc/kubernetes/pki/apiserver-etcd-client.crt \
  --from-file=apiserver-etcd-client.key=/etc/kubernetes/pki/apiserver-etcd-client.key
```



### 租户集群

### namespace

```shell
kubectl create ns [tenant]
```

### 拷贝etcd证书

```shell
kubectl create secret generic etcd-cert \
  -n [tenant] \
  --from-file=ca.crt=/etc/kubernetes/pki/etcd/ca.crt \
  --from-file=apiserver-etcd-client.crt=/etc/kubernetes/pki/apiserver-etcd-client.crt \
  --from-file=apiserver-etcd-client.key=/etc/kubernetes/pki/apiserver-etcd-client.key
```

#### 证书

| cert                          | CN                             | Organization   | KeyUsage                                                     | ExtendKeyUsage                | DNS                                                          | IPs                                      |
| ----------------------------- | ------------------------------ | -------------- | ------------------------------------------------------------ | ----------------------------- | ------------------------------------------------------------ | ---------------------------------------- |
| ca                            | kcp                            | -              | Digital Signature<br />Key Encipherment<br />Certificate Sign | -                             | -                                                            | -                                        |
| apiserver                     | kube-apiserver                 | -              | Digital Signature<br />Key Encipherment                      | TLS Web Server Authentication | host-control-plane<br />kubernetes<br />kubernetes.default<br />kubernetes.default.svc,<br />kubernetes.default.svc.cluster.local<br />localhost | 10.96.0.1<br />172.18.0.2<br />127.0.0.1 |
| apiserver-kubelet-client      | kube-apiserver-kubelet-client  | system:masters | Digital Signature<br />Key Encipherment                      | TLS Web Client Authentication | -                                                            | -                                        |
| front-proxy-ca                | front-proxy-ca                 | -              | Digital Signature<br />Key Encipherment<br />Certificate Sign | -                             | front-proxy-ca                                               | -                                        |
| front-proxy-client            | front-proxy-client             | -              | Digital Signature<br />Key Encipherment                      | TLS Web Client Authentication | -                                                            | -                                        |
| sa: 密钥对                    | -                              | -              | -                                                            | -                             | -                                                            | -                                        |
| kubeconfig-admin              | kubernetes-admin               | system:masters | Digital Signature<br />Key Encipherment                      | TLS Web Client Authentication | -                                                            | -                                        |
| kubeconfig-controller-manager | system:kube-controller-manager | -              | Digital Signature<br />Key Encipherment                      | TLS Web Client Authentication | -                                                            | -                                        |

```shell
kubectl create secret generic server-cert \
  -n [tenant] \
  --from-file=ca.crt=/etc/kubernetes/pki/server/ca.crt \
  --from-file=ca.key=/etc/kubernetes/pki/server/ca.key \
  --from-file=apiserver.crt=/etc/kubernetes/pki/server/apiserver.crt \
  --from-file=apiserver.key=/etc/kubernetes/pki/server/apiserver.key \
  --from-file=apiserver-kubelet-client.crt=/etc/kubernetes/pki/server/apiserver-kubelet-client.crt \
  --from-file=apiserver-kubelet-client.key=/etc/kubernetes/pki/server/apiserver-kubelet-client.key \
  --from-file=front-proxy-ca.crt=/etc/kubernetes/pki/server/front-proxy-ca.crt \
  --from-file=front-proxy-ca.key=/etc/kubernetes/pki/server/front-proxy-ca.key \
  --from-file=front-proxy-client.crt=/etc/kubernetes/pki/server/front-proxy-client.crt \
  --from-file=front-proxy-client.key=/etc/kubernetes/pki/server/front-proxy-client.key \
  --from-file=sa.pub=/etc/kubernetes/pki/server/sa.pub \
  --from-file=sa.key=/etc/kubernetes/pki/server/sa.key

kubectl create secret generic kubeconfig-admin \
  -n [tenant] \
  --from-file=admin.conf=/etc/kubernetes/kubeconfig/admin.conf

kubectl create secret generic kubeconfig-controller-manager \
  -n [tenant] \
  --from-file=controller-manager.conf=/etc/kubernetes/kubeconfig/controller-manager.conf
```

#### 部署

##### apiserver

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-apiserver
  namespace: [tenant]
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-apiserver
      tenant: [tenant]
  template:
    metadata:
      labels:
        app: kube-apiserver
        tenant: [tenant]
    spec:
      containers:
      - name: apiserver
        image: k8s.gcr.io/kube-apiserver:v1.23.4
        command:
        - kube-apiserver
        - --advertise-address=0.0.0.0
				- --allow-privileged=true
				- --authorization-mode=Node,RBAC
				- --client-ca-file=/etc/kubernetes/pki/ca.crt
				- --enable-admission-plugins=NodeRestriction
				- --enable-bootstrap-token-auth=true
				- --etcd-cafile=/etc/kubernetes/pki/etcd/ca.crt
				- --etcd-certfile=/etc/kubernetes/pki/etcd/apiserver-etcd-client.crt
				- --etcd-keyfile=/etc/kubernetes/pki/etcd/apiserver-etcd-client.key
				- --etcd-servers=https://[etcd]:2379
				- --etcd-prefix=/[tenant]/registry
				- --insecure-port=0
				- --kubelet-client-certificate=/etc/kubernetes/pki/server/apiserver-kubelet-client.crt
				- --kubelet-client-key=/etc/kubernetes/pki/server/apiserver-kubelet-client.key
				- --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname
				- --proxy-client-cert-file=/etc/kubernetes/pki/server/front-proxy-client.crt
				- --proxy-client-key-file=/etc/kubernetes/pki/server/front-proxy-client.key
				- --requestheader-allowed-names=front-proxy-client
				- --requestheader-client-ca-file=/etc/kubernetes/pki/server/front-proxy-ca.crt
				- --requestheader-extra-headers-prefix=X-Remote-Extra-
				- --requestheader-group-headers=X-Remote-Group
				- --requestheader-username-headers=X-Remote-User
				- --secure-port=6443
				- --service-account-key-file=/etc/kubernetes/pki/server/sa.pub
				- --service-cluster-ip-range=10.101.0.0/16
				- --tls-cert-file=/etc/kubernetes/pki/server/apiserver.crt
				- --tls-private-key-file=/etc/kubernetes/pki/server/apiserver.key
				volumeMounts:
				- name: etcd-cert
				  mountPath: /etcd/kubernetes/pki/etcd
				  readOnly: true
				- name: server-cert
				  mountPath: /etc/kubernetes/pki/server
				  readOnly: true
				livenessProbe:
          failureThreshold: 8
          httpGet:
            host: 127.0.0.1
            path: /livez
            port: 6443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
        readinessProbe:
          failureThreshold: 3
          httpGet:
            host: 127.0.0.1
            path: /readyz
            port: 6443
            scheme: HTTPS
          periodSeconds: 1
          successThreshold: 1
          timeoutSeconds: 15
        startupProbe:
          failureThreshold: 24
          httpGet:
            host: 127.0.0.1
            path: /livez
            port: 6443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
      volumes:
      - name: etcd-cert
        secret:
          secretName: etcd-cert
      - name: server-cert
        secret:
          secretName: server-cert
---
apiVersion: core/v1
kind: Service
metadata:
  name: kube-apiserver
  namespace: [tenant]
spec:
  selector:
    app: kube-apiserver
    tenant: [tenant]
  ports:
  - name: https
    protocol: TCP
    port: 6443
    targetPort: 6443
```

##### controller-manager

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-controller-manager
  namespace: [tenant]
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-controller-manager
      tenant: [tenant]
  template:
    metadata:
      labels:
        app: kube-controller-manager
        tenant: [tenant]
    spec:
      containers:
      - name: controller-manager
        image: k8s.gcr.io/kube-controller-manager:v1.23.4
        command:
        - kube-controller-manager
        - --allocate-node-cidrs=true
        - --authentication-kubeconfig=/etc/kubernetes/kubeconfig/controller-manager.conf
        - --authorization-kubeconfig=/etc/kubernetes/kubeconfig/controller-manager.conf
        - --bind-address=0.0.0.0
        - --client-ca-file=/etc/kubernetes/pki/ca.crt
        - --cluster-cidr=10.100.0.0/16
        - --cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt
        - --cluster-signing-key-file=/etc/kubernetes/pki/ca.key
        - --controllers=*,bootstrapsigner,tokencleaner
        - --kubeconfig=/etc/kubernetes/kubeconfig/controller-manager.conf
        - --leader-elect=true
        - --node-cidr-mask-size=24
        - --requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt
        - --root-ca-file=/etc/kubernetes/pki/ca.crt
        - --service-account-private-key-file=/etc/kubernetes/pki/sa.key
        - --use-service-account-credentials=true
				volumeMounts:
				- name: server-cert
				  mountPath: /etc/kubernetes/pki
				  readOnly: true
				- name: kubeconfig
				  mountPath: /etc/kubernetes/kubeconfig
				  readOnly: true
				livenessProbe:
          failureThreshold: 8
          httpGet:
            host: 127.0.0.1
            path: /livez
            port: 6443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
        readinessProbe:
          failureThreshold: 3
          httpGet:
            host: 127.0.0.1
            path: /readyz
            port: 6443
            scheme: HTTPS
          periodSeconds: 1
          successThreshold: 1
          timeoutSeconds: 15
        startupProbe:
          failureThreshold: 24
          httpGet:
            host: 127.0.0.1
            path: /livez
            port: 6443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
      volumes:
      - name: kubeconfig
        secret:
          secretName: kubeconfig-controller-manager
      - name: server-cert
        secret:
          secretName: server-cert
```



### 验证



## 2. CRD 设计




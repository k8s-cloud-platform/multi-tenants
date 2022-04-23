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
  --from-file=etcd-ca.crt=/etc/kubernetes/pki/etcd/ca.crt \
  --from-file=apiserver-etcd-client.crt=/etc/kubernetes/pki/apiserver-etcd-client.crt \
  --from-file=apiserver-etcd-client.key=/etc/kubernetes/pki/apiserver-etcd-client.key
```

### 租户集群

#### namespace

```shell
kubectl create ns [tenant]
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
  --type=kcp/kube-secret \
  --from-file=etcd-ca.crt=/etc/kubernetes/pki/etcd/ca.crt \
  --from-file=apiserver-etcd-client.crt=/etc/kubernetes/pki/apiserver-etcd-client.crt \
  --from-file=apiserver-etcd-client.key=/etc/kubernetes/pki/apiserver-etcd-client.key
  --from-file=ca.crt=examples/pki/apiserver-ca.crt \
  --from-file=ca.key=examples/pki/apiserver-ca.key \
  --from-file=apiserver.crt=examples/pki/apiserver.crt \
  --from-file=apiserver.key=examples/pki/apiserver.key \
  --from-file=apiserver-kubelet-client.crt=examples/pki/apiserver-kubelet-client.crt \
  --from-file=apiserver-kubelet-client.key=examples/pki/apiserver-kubelet-client.key \
  --from-file=front-proxy-ca.crt=examples/pki/front-proxy-ca.crt \
  --from-file=front-proxy-ca.key=examples/pki/front-proxy-ca.key \
  --from-file=front-proxy-client.crt=examples/pki/front-proxy-client.crt \
  --from-file=front-proxy-client.key=examples/pki/front-proxy-client.key \
  --from-file=sa.pub=examples/pki/sa.pub \
  --from-file=sa.key=examples/pki/sa.key

kubectl create secret generic kubeconfig-admin \
  -n [tenant] \
  --type=kcp/kubeconfig \
  --from-file=admin.conf=examples/kubeconfig/admin.conf

kubectl create secret generic kubeconfig-controller-manager \
  -n [tenant] \
  --type=kcp/kubeconfig \
  --from-file=controller-manager.conf=examples/kubeconfig/controller-manager.conf
```

#### 部署

##### apiserver

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-apiserver
  namespace: [ tenant ]
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-apiserver
      tenant: [ tenant ]
  template:
    metadata:
      labels:
        app: kube-apiserver
        tenant: [ tenant ]
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
            - --etcd-cafile=/etc/kubernetes/pki/etcd-ca.crt
            - --etcd-certfile=/etc/kubernetes/pki/apiserver-etcd-client.crt
            - --etcd-keyfile=/etc/kubernetes/pki/apiserver-etcd-client.key
            - --etcd-servers=https://[etcd]:2379
            - --etcd-prefix=/[tenant]/registry
            - --insecure-port=0
            - --kubelet-client-certificate=/etc/kubernetes/pki/apiserver-kubelet-client.crt
            - --kubelet-client-key=/etc/kubernetes/pki/apiserver-kubelet-client.key
            - --kubelet-preferred-address-types=InternalIP,ExternalIP,Hostname
            - --proxy-client-cert-file=/etc/kubernetes/pki/front-proxy-client.crt
            - --proxy-client-key-file=/etc/kubernetes/pki/front-proxy-client.key
            - --requestheader-allowed-names=front-proxy-client
            - --requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt
            - --requestheader-extra-headers-prefix=X-Remote-Extra-
            - --requestheader-group-headers=X-Remote-Group
            - --requestheader-username-headers=X-Remote-User
            - --secure-port=6443
            - --service-account-issuer=https://kubernetes.default.svc.cluster.local
            - --service-account-key-file=/etc/kubernetes/pki/sa.pub
            - --service-account-signing-key-file=/etc/kubernetes/pki/sa.key
            - --service-cluster-ip-range=10.101.0.0/16
            - --tls-cert-file=/etc/kubernetes/pki/apiserver.crt
            - --tls-private-key-file=/etc/kubernetes/pki/apiserver.key
          volumeMounts:
            - name: server-cert
              mountPath: /etc/kubernetes/pki
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
        - name: server-cert
          secret:
            secretName: server-cert
---
apiVersion: v1
kind: Service
metadata:
  name: kube-apiserver
  namespace: [ tenant ]
spec:
  selector:
    app: kube-apiserver
    tenant: [ tenant ]
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
  namespace: [ tenant ]
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kube-controller-manager
      tenant: [ tenant ]
  template:
    metadata:
      labels:
        app: kube-controller-manager
        tenant: [ tenant ]
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
        - name: server-cert
          secret:
            secretName: server-cert
        - name: kubeconfig
          secret:
            secretName: kubeconfig-controller-manager
```

### 部署结果

```shell
$ kubectl get deploy -n tenant-1
NAME                      READY   UP-TO-DATE   AVAILABLE   AGE
kube-apiserver            1/1     1            1           12h
kube-controller-manager   1/1     1            1           3m56s

$ kubectl get svc -n tenant-1
NAME             TYPE       CLUSTER-IP      EXTERNAL-IP   PORT(S)          AGE
kube-apiserver   NodePort   10.96.117.146   <none>        6443:30140/TCP   12h

$ kubectl get secret -n tenant-1
NAME                            TYPE             DATA   AGE
etcd-cert                       kcp/etcd-secret  3      23h
kubeconfig-admin                kcp/kubeconfig   1      10m
kubeconfig-controller-manager   kcp/kubeconfig   1      9m46s
server-cert                     kcp/kube-secret  12     12h
```

### 验证

部署测试容器，进入容器内执行kubectl命令验证

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: kubectl
  namespace: tenant-1
spec:
  containers:
    - name: kubectl
      image: bitnami/kubectl:1.23.4
      imagePullPolicy: IfNotPresent
      command:
        - sleep
        - "6000"
      volumeMounts:
        - name: kubeconfig
          mountPath: /etc/kubernetes/kubeconfig
          readOnly: true
  volumes:
    - name: kubeconfig
      secret:
        secretName: kubeconfig-admin
```

## 2. CRD 设计




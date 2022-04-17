# tenant design



## 准备工作

将 etcd 相关的证书存放到 kube-system/etcd-cert secret 中

- ca.crt: /etc/kubernetes/pki/etcd/ca.crt

- apiserver-etcd-client.crt: /etc/kubernetes/pki/apiserver-etcd-client.crt
- apiserver-etcd-client.key: /etc/kubernetes/pki/apiserver-etcd-client.key

将 kubeconfig 存放到 kube-system/kubeconfig secret 中



## Certification

签发相关证书

- ca.crt: /etc/kubernetes/pki/ca.crt
- apiserver.crt: /etc/kubernetes/pki/apiserver.crt
- apiserver.key: /etc/kubernetes/pki/apiserver.key
- apiserver-kubelet-client.crt: /etc/kubernetes/pki/apiserver-kubelet-client.crt

- apiserver-kubelet-client.key: /etc/kubernetes/pki/apiserver-kubelet-client.key
- front-proxy-client.crt: /etc/kubernetes/pki/front-proxy-client.crt
- front-proxy-client.key: /etc/kubernetes/pki/front-proxy-client.key
- sa.pub: /etc/kubernetes/pki/sa.pub
- sa.key: /etc/kubernetes/pki/sa.key



## Deployment、Service

### apiserver

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-apiserver
  namespace: [tenant]
  labels:
    app: kube-apiserver
    tenant: [tenant]
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
				- --service-account-key-file=/etc/kubernetes/pki/sa.pub
				- --service-cluster-ip-range=10.101.0.0/16
				- --tls-cert-file=/etc/kubernetes/pki/apiserver.crt
				- --tls-private-key-file=/etc/kubernetes/pki/apiserver.key
				livenessProbe:
          failureThreshold: 8
          httpGet:
            host: 172.18.0.2
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
            host: 172.18.0.2
            path: /readyz
            port: 6443
            scheme: HTTPS
          periodSeconds: 1
          successThreshold: 1
          timeoutSeconds: 15
        startupProbe:
          failureThreshold: 24
          httpGet:
            host: 172.18.0.2
            path: /livez
            port: 6443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
        resources:
          requests:
            cpu: 250m
---
apiVersion: core/v1
kind: Service
metadata:
  name: kube-apiserver
  namespace: [tenant]
  labels:
    app: kube-apiserver
    tenant: [tenant]
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



### controller-manager

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kube-controller-manager
  namespace: [tenant]
  labels:
    app: kube-controller-manager
    tenant: [tenant]
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
        - --authentication-kubeconfig=/etc/kubernetes/controller-manager.conf
        - --authorization-kubeconfig=/etc/kubernetes/controller-manager.conf
        - --bind-address=0.0.0.0
        - --client-ca-file=/etc/kubernetes/pki/ca.crt
        - --cluster-cidr=10.100.0.0/16
        - --cluster-signing-cert-file=/etc/kubernetes/pki/ca.crt
        - --cluster-signing-key-file=/etc/kubernetes/pki/ca.key
        - --controllers=*,bootstrapsigner,tokencleaner
        - --kubeconfig=/etc/kubernetes/controller-manager.conf
        - --leader-elect=true
        - --node-cidr-mask-size=24
        - --requestheader-client-ca-file=/etc/kubernetes/pki/front-proxy-ca.crt
        - --root-ca-file=/etc/kubernetes/pki/ca.crt
        - --service-account-private-key-file=/etc/kubernetes/pki/sa.key
        - --use-service-account-credentials=true
				livenessProbe:
          failureThreshold: 8
          httpGet:
            host: 172.18.0.2
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
            host: 172.18.0.2
            path: /readyz
            port: 6443
            scheme: HTTPS
          periodSeconds: 1
          successThreshold: 1
          timeoutSeconds: 15
        startupProbe:
          failureThreshold: 24
          httpGet:
            host: 172.18.0.2
            path: /livez
            port: 6443
            scheme: HTTPS
          initialDelaySeconds: 10
          periodSeconds: 10
          successThreshold: 1
          timeoutSeconds: 15
        resources:
          requests:
            cpu: 250m
```


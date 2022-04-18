/*
Copyright 2022 The KCP Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package kubeconfig

import (
	"crypto"
	"crypto/x509"
	"fmt"

	"k8s.io/client-go/tools/clientcmd/api"
	certutil "k8s.io/client-go/util/cert"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/secret"
)

// New creates a new Kubeconfig using the cluster name and specified endpoint.
func New(clusterName, endpoint string, caCert *x509.Certificate, caKey crypto.Signer) (*api.Config, error) {
	cfg := &secret.CertsConfig{
		Config: certutil.Config{
			CommonName:   "kubernetes-admin",
			Organization: []string{"system:masters"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}

	cert, key, err := secret.NewCertAndKey(caCert, caKey, cfg)
	if err != nil {
		return nil, err
	}

	userName := fmt.Sprintf("%s-admin", clusterName)
	contextName := fmt.Sprintf("%s@%s", userName, clusterName)

	return &api.Config{
		Clusters: map[string]*api.Cluster{
			clusterName: {
				Server:                   endpoint,
				CertificateAuthorityData: secret.EncodeCertPEM(caCert),
			},
		},
		Contexts: map[string]*api.Context{
			contextName: {
				Cluster:  clusterName,
				AuthInfo: userName,
			},
		},
		AuthInfos: map[string]*api.AuthInfo{
			userName: {
				ClientKeyData:         secret.EncodePrivateKeyPEM(key),
				ClientCertificateData: secret.EncodeCertPEM(cert),
			},
		},
		CurrentContext: contextName,
	}, nil
}

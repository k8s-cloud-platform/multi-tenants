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

// NewWithSecret creates a new kubeconfig using the cluster name and specified endpoint.
func NewWithSecret(clusterName, endpoint string, caCert *x509.Certificate, caKey crypto.Signer, config *certutil.Config) (*api.Config, error) {
	cert, key, err := secret.NewCertAndKey(caCert, caKey, config)
	if err != nil {
		return nil, err
	}

	userName := fmt.Sprintf("%s-%s", clusterName, config.CommonName)
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

// NewWithToken creates a new kubeconfig using the cluster name and specified endpoint.
func NewWithToken(clusterName, endpoint string, caCert *x509.Certificate, token string, config *certutil.Config) (*api.Config, error) {
	userName := fmt.Sprintf("%s-%s", clusterName, config.CommonName)
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
				Token: token,
			},
		},
		CurrentContext: contextName,
	}, nil
}

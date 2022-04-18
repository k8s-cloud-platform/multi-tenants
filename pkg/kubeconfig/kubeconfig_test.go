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
	"crypto/x509"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/secret"
)

func TestWithSecret(t *testing.T) {
	caCert, caKey, err := secret.NewCA()
	assert.NoError(t, err)

	cfg := &secret.CertsConfig{
		Config: certutil.Config{
			CommonName:   "kubernetes-admin",
			Organization: []string{"system:masters"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}

	c, err := NewWithSecret("tenant-cluster", "127.0.0.1:6443", caCert, caKey, cfg)
	assert.NoError(t, err)

	config, err := clientcmd.Write(*c)
	assert.NoError(t, err)
	t.Logf("%s", config)
}

func TestWithToken(t *testing.T) {
	caCert, _, err := secret.NewCA()
	assert.NoError(t, err)

	cfg := &secret.CertsConfig{
		Config: certutil.Config{
			CommonName:   "kubernetes-token",
			Organization: []string{"system:token"},
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
		},
	}

	c, err := NewWithToken("tenant-cluster", "127.0.0.1:6443", caCert, "token123", cfg)
	assert.NoError(t, err)

	config, err := clientcmd.Write(*c)
	assert.NoError(t, err)
	t.Logf("%s", config)
}

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

package secret

import (
	"crypto/x509"
	"fmt"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	certutil "k8s.io/client-go/util/cert"
)

type testCase struct {
	name      string
	caConfig  *certutil.Config
	certCases []certCase
}

type certCase struct {
	certName   string
	certConfig *certutil.Config
}

func TestCerts(t *testing.T) {
	tests := []testCase{
		{
			name: "apiserver",
			certCases: []certCase{
				{
					certName: "apiserver",
					certConfig: &certutil.Config{
						CommonName: "kube-apiserver",
						AltNames: certutil.AltNames{
							DNSNames: []string{
								"kube-apiserver.tenant-1.svc",
								"localhost",
							},
							IPs: []net.IP{
								net.ParseIP("127.0.0.1"),
							},
						},
						Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
					},
				},
				{
					certName: "apiserver-kubelet-client",
					certConfig: &certutil.Config{
						CommonName:   "kube-apiserver-kubelet-client",
						Organization: []string{"system:masters"},
						Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
					},
				},
				{
					certName: "kubeconfig-admin",
					certConfig: &certutil.Config{
						CommonName:   "kubernetes-admin",
						Organization: []string{"system:masters"},
						Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
					},
				},
				{
					certName: "kubeconfig-controller-manager",
					certConfig: &certutil.Config{
						CommonName: "system:kube-controller-manager",
						Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
					},
				},
			},
		},
		{
			name: "front-proxy",
			caConfig: &certutil.Config{
				AltNames: certutil.AltNames{
					DNSNames: []string{"front-proxy-ca"},
				},
			},
			certCases: []certCase{
				{
					certName: "front-proxy-client",
					certConfig: &certutil.Config{
						CommonName: "front-proxy-client",
						Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Logf("----- sign certs for: %s", test.name)

		t.Logf("----- sign ca cert")
		ca, key, err := NewCA(test.caConfig)
		assert.NoError(t, err)
		fmt.Printf("%s\n", EncodeCertPEM(ca))
		fmt.Printf("%s\n", EncodePrivateKeyPEM(key))

		for _, cert := range test.certCases {
			t.Logf("----- sign certs: %s", cert.certName)
			cert, key, err := NewCertAndKey(ca, key, cert.certConfig)
			assert.NoError(t, err)
			fmt.Printf("%s\n", EncodeCertPEM(cert))
			fmt.Printf("%s\n", EncodePrivateKeyPEM(key))
		}
	}
}

func TestPubAndKey(t *testing.T) {
	t.Log("----- sign for sa")

	pub, key, err := NewPubAndKey()
	assert.NoError(t, err)

	encodedPub, err := EncodePublicKeyPEM(pub)
	assert.NoError(t, err)

	fmt.Printf("%s\n", encodedPub)
	fmt.Printf("%s\n", EncodePrivateKeyPEM(key))
}

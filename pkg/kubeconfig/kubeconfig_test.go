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
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"
	certutil "k8s.io/client-go/util/cert"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/secret"
)

const (
	ca = `-----BEGIN CERTIFICATE-----
MIIC9jCCAd6gAwIBAgIBADANBgkqhkiG9w0BAQsFADAbMQwwCgYDVQQKEwNrY3Ax
CzAJBgNVBAMTAmNhMB4XDTIyMDQyMTAwMTkxNVoXDTMyMDQxODAwMjQxNVowGzEM
MAoGA1UEChMDa2NwMQswCQYDVQQDEwJjYTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBANXy1NfUnYe+x9avl7vac2/MxMklylagA7GobUzDDBYlmVcc6/hf
nvZxyB6DBTQgNr7vdyudumCp4C7deYryKzk6knJ+sEZ/AENNgJS0KLd7e2OxMT/x
6FGxSZjxWf8sHxDIOS0F/zs6XzyOYD4g+RFHTk6nBXskG+/ikoaFZZf5bMBLMuz/
+h8GBzw0n4W0OeUSMKkgsutsJehH3YZ117zquqO2kA7Mm14CnQL6sRIl9zy44Clt
odDNjvIhKWWlSqQ+Nkf4QRBxd8qQBIM5znE8adNaw+P2NYiFuORlDqPRPV1Cok6F
bgzp9iMJf7/Pk8aWE/5BYPFkmw/NyJ/1xGMCAwEAAaNFMEMwDgYDVR0PAQH/BAQD
AgKkMBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0OBBYEFMRuajiQ9SAWj/x76P9e
bxi/xtdgMA0GCSqGSIb3DQEBCwUAA4IBAQC3c9RsOsRcuBRDjlCPeJykKpHItWSi
15uhhAiKBVj0iKgt+5SErtDGuSIa2bY2LSzQi79W+mau3Ga5X2AYEyYdu0ymxiDH
SagY+e/L/PxrIo22yHGIYxE2VK1Hty6KFMKtSFjB7fZscysA1tIYa4+qNxJungvc
7jDYXwbrsP+FDc/IM3T5ZlIG6A2nHE1qCPal+n7AxhlwjOG8Iby8ep7CDmNP3Zc5
QkMRTXdbFsQ2fFBkthquB0CjfIb1OvO1Pz4Qok5i+WeHzpDvQmDv03E+93pMV+gy
aAlIEpGHa0zlNJaIm+HqNOR+oc6KRUNn8HBPCV9IVutcsgipXfAXQKU4
-----END CERTIFICATE-----`
	key = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA1fLU19Sdh77H1q+Xu9pzb8zEySXKVqADsahtTMMMFiWZVxzr
+F+e9nHIHoMFNCA2vu93K526YKngLt15ivIrOTqScn6wRn8AQ02AlLQot3t7Y7Ex
P/HoUbFJmPFZ/ywfEMg5LQX/OzpfPI5gPiD5EUdOTqcFeyQb7+KShoVll/lswEsy
7P/6HwYHPDSfhbQ55RIwqSCy62wl6EfdhnXXvOq6o7aQDsybXgKdAvqxEiX3PLjg
KW2h0M2O8iEpZaVKpD42R/hBEHF3ypAEgznOcTxp01rD4/Y1iIW45GUOo9E9XUKi
ToVuDOn2Iwl/v8+TxpYT/kFg8WSbD83In/XEYwIDAQABAoIBAQCGtU6uVoCZZ9YL
pqOy8+ibDCMbQ2ATCs1InvOy0Vxa1XGnF967k/lS0nFeRMCSAXZ24e/21mjzVAKD
f493nOL6NZbf4ES9HlncBoBfINBCNs2KB5cq2/Wa+jWMxuoTcIRe3LKjVpNNh4NP
bZXLiJdJ+cukWiVpU2MDt3TqnjNJa+uo0Tq/oMp9+gcVYIcSG0oG+OiCattLS082
zlIbHVS1dnFJD26T1t0wFGRgP/w4xQkL06p9HnOervGVDbOL7+oJFLyPi5PzThKt
0k9iNm4cIYRmourqHrytALg6XqzvlSWHqY7c/6KwsLknFSmDnnWAJh4h8CbkTWeC
k9YlLbeBAoGBAPYMylA/UF7c2Ov54mW3dLUXISv350NMTaJl1KDv1X8T0DAZOyAb
vA1/KctGu+lxeF+vsyAikIUkytw4fwVruZK5vdWZ5jNpP0E0I06ZLY6+157ep9Ez
L4aRqGRk64FFW/poqL9/nqslpgIPRslTEe3BHvdmDyGmSlPPeH34937NAoGBAN6Z
ttzRDSYiroOwDyVXUOnwkO9HyS72h3gUx44fz8uXEV35jaeEQMot8zIXV4SrfXAj
U9TEmYJV1Ygdy2F1rjzGFFgd48ZjldYhEIMpo7j7Doea0gH+1iBZcvyUxgn0h/2X
Gcu9Cxf4UUr7v9SL3Co/qSarXV86f8BaRQ69me/vAoGBAPOiypoXd3/XczyG3TUX
KYs/cetVuDudTKpn5tnNYjxI5DccZL1weQ9sQXg1fFC1uRoWompJj40nV3AkyTIW
71Roc58UmyuBJc9szyE+8CuOsXdwxf+SjZbUmEeA3unMQj18cMvelMCnsPANLGyh
wvkH55/WAVmyjZrbr1YdtwRpAoGAQJwBFYqf9Gewm4xoeQAk/iHTD4CYhkcL/vCZ
E0lseL06Tzp1wR+D5qr7oZhg0KMt33SBioodvFTJi56KP5l2nzEnCoRybkzNWr9y
GX/mg3RrPTTUq7uZzv3ngCR2bsFJD3PiCt6t76Elh0wW28xUdPbRU+X9Tfyew+fv
j1285oUCgYAYGttc6HdmXirsSHshLSZ+tJlAqtFzEIX1pT/O0eSsqQqfWxIZvT4A
fcmOWHNtB5/7heyev6dtNexmS3E0lhER7+2XYK5ZTH+A0je72mW3+qQhHWVrr7n3
5KETYz8VP78j5oTEgn6FVf2myTSIKL3StCn2t3Dvdhxowa18NvKHHA==
-----END RSA PRIVATE KEY-----`
	saveFile = false
)

type testCase struct {
	name   string
	config *certutil.Config
	token  string
}

func TestWithSecret(t *testing.T) {
	tests := []testCase{
		{
			name: "admin.conf",
			config: &certutil.Config{
				CommonName:   "kubernetes-admin",
				Organization: []string{"system:masters"},
				Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		},
		{
			name: "controller-manager.conf",
			config: &certutil.Config{
				CommonName: "system:kube-controller-manager",
				Usages:     []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			},
		},
	}

	caCert, err := secret.DecodeCertPEM([]byte(ca))
	assert.NoError(t, err)
	caKey, err := secret.DecodePrivateKeyPEM([]byte(key))
	assert.NoError(t, err)

	for _, test := range tests {
		t.Logf("----- sign certs for: %s", test.name)

		c, err := NewWithSecret("tenant-1", "https://kube-apiserver.tenant-1.svc:6443", caCert, caKey, test.config)
		assert.NoError(t, err)
		config, err := clientcmd.Write(*c)
		assert.NoError(t, err)
		t.Logf("%s", config)

		if saveFile {
			ioutil.WriteFile(test.name, config, 0644)
		}
	}
}

func TestWithToken(t *testing.T) {
	tests := []testCase{
		{
			name:  "test.conf",
			token: "test123",
			config: &certutil.Config{
				CommonName: "test",
			},
		},
	}

	caCert, err := secret.DecodeCertPEM([]byte(ca))
	assert.NoError(t, err)

	for _, test := range tests {
		t.Logf("----- sign certs for: %s", test.name)

		c, err := NewWithToken("tenant-1", "https://kube-apiserver.tenant-1.svc:6443", caCert, test.token, test.config)
		assert.NoError(t, err)
		config, err := clientcmd.Write(*c)
		assert.NoError(t, err)
		t.Logf("%s", config)

		if saveFile {
			ioutil.WriteFile(test.name, config, 0644)
		}
	}
}

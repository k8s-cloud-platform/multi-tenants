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
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	certutil "k8s.io/client-go/util/cert"
)

var (
	caCertByte = `-----BEGIN CERTIFICATE-----
MIIC9jCCAd6gAwIBAgIBADANBgkqhkiG9w0BAQsFADAbMQwwCgYDVQQKEwNrY3Ax
CzAJBgNVBAMTAmNhMB4XDTIyMDQxODEzNTgyM1oXDTMyMDQxNTE0MDMyM1owGzEM
MAoGA1UEChMDa2NwMQswCQYDVQQDEwJjYTCCASIwDQYJKoZIhvcNAQEBBQADggEP
ADCCAQoCggEBAKYEBDRXEJWEgRYTn2uB0hz/xt5tOb6xQD4yvIrMA83X/TPivb6Q
Hh2f0aFbSfJIskCQkTC1+XsGq60AJh8FZPJ/d6UjrEnMDBnF+spgFjFA485SdRTg
IQBKMivw/X1Pxz1RikR74ievvR/8bHB9tAkD6sXDUa8k1fSSX991lk9Ft3oFTIUS
YdGlNc/Pi5Q5IxRyq23hz720XCbAHggbjDYmb/TlzT8uOTvfsPJw+a4HH3I3VrxP
zuC7ue8KXZxUo3wF3I7UYtl9qOG9f5nviFqDacFOiTzj9rlO33GRYOCl7mdvZf6x
sSYQL36RJuoewKmnfr4YUIwEA+T070pYKh8CAwEAAaNFMEMwDgYDVR0PAQH/BAQD
AgKkMBIGA1UdEwEB/wQIMAYBAf8CAQAwHQYDVR0OBBYEFK4eRH+GPcpBr+cSO9fc
qkeUwb4aMA0GCSqGSIb3DQEBCwUAA4IBAQBRFDcx3hr3mrFaK/7/nBBefVdann9X
gtPixOHppkLsipxQGjF4B6qiGBHkNOmc0yx5T1faTsz+rZCR/2dWmmR8h5iOiki5
so5/BPg8KD4M993Wkd6MPpF0YRWBZ+4rsl4cGoxkJGgtOx8gktaNkHIYAro42WVu
0taJkBPYZQguEiZz9UskTlj8E6nJrKW6F5rXAAwljIYt6+7CQkde9MlS3XkRgJBl
5fqv+dYmaMQvSrHylZ4DXxClfQbtNtAlfnMXt1pChbz3nS4OAwleqAoN0e0B83pd
eoMHQuCcTD/zjHXlvPUvpjJiJ3sLuRo7/MUAV3yXYcwsOCimWpBfy+AS
-----END CERTIFICATE-----`
	caKeyByte = `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEApgQENFcQlYSBFhOfa4HSHP/G3m05vrFAPjK8iswDzdf9M+K9
vpAeHZ/RoVtJ8kiyQJCRMLX5ewarrQAmHwVk8n93pSOsScwMGcX6ymAWMUDjzlJ1
FOAhAEoyK/D9fU/HPVGKRHviJ6+9H/xscH20CQPqxcNRryTV9JJf33WWT0W3egVM
hRJh0aU1z8+LlDkjFHKrbeHPvbRcJsAeCBuMNiZv9OXNPy45O9+w8nD5rgcfcjdW
vE/O4Lu57wpdnFSjfAXcjtRi2X2o4b1/me+IWoNpwU6JPOP2uU7fcZFg4KXuZ29l
/rGxJhAvfpEm6h7Aqad+vhhQjAQD5PTvSlgqHwIDAQABAoIBAQCAwL6uBRQUkZY3
k/Jgw2c8HFaUrKtLLFbBpkG5d24/15JFCkXUJBtnKErBVHZuFFlCX5xq5cbd1heQ
7XujNWDL/XXCOn9lIH4GAxh0mb68ZjIHEsZA8W5GtkRg9d7yr8u6z6FnaZjE5LPN
ucw0FhlpoIMU611Pc0cIDfmS6bQwN4aoVht8Q7eJBzqRugAjk/2ZZwwq26/DGuxL
lWS6iwQALU9otK8n2ZVC/sVr62MVUTz3iYPvKvAmChkupy+Skhw5UVdHDEXUm9oY
hKSj10iBCgc1T0y28eaXvbI2IDRiwarF+5nxwmL4q6eJmHPljhqSP01tXhbK27ro
c8py/HYZAoGBAMir1opBviPwYp0czFEVjV1eG5ooXu37YsXbdaEUTyC4oHTSWUOy
lfWBY7YnOojGwOdfLYzmt5iXGlmcqfvEzKLuiR+s4CQsntsfoeNC2jqUSxG1A/8q
4THrv9iYkJ3MyCMwm+kfQOK0MXTiWRC1bbsZpkTNDfKLbUYCT+BUVznTAoGBANPK
DcJjFZs9yQ7dh42cyON3kdAFinvxf73He3aciLgNWDWpR7ECTbnZ4gQT9q/B/wD5
XSDR0+kPY80zg8tGt8VsvvPFQyZ4MryXYLvFkD9sUpBPq2PJBBXuR9fpYAsptx1H
XV9mvBDoodG1DhkwAJiGw1optz7vrZMBzaJxBTMFAoGAHOFeRxefHd0C1EnIvgK0
DbP1lQIqZ2W+mWs0SrJ5kOQBc86yOiQBoQu2JgkPS5APQz1XeOfblIJqsGrzq+Bu
8yQRaBWhTJN/aVnsGqEMd1HQXAQJRzgMoPqk6a9LYOqQA0CDr9FnrTjxxTiuSNFT
sZHHdC9uynF2kJTUmhyJUTECgYEA0dz9R2sB5QQTcHyKLwR2eBhqz7Q95tWGnFrI
d99jBuaKSxpEJR3AQXKrKRlTMBRPCPsQkk35647JeXQbm9mmjqZUjaAiC0O+Guzx
+P8rEf3XzpJzpfxx5P1qCYSDHGyovAWUe82EfEjY2MHNh53uaTXY7EY8A8xitjJq
PxY5bbUCgYBeab4zDorSpOjeRs2HNkgTplIfcq0t/wzCC00snaX4azweP80ZHnMr
DlXF/qGOaLvstH2VzG51McMhh4tPA57D/rM6org3YJ8Osf7ycz8aS37BycJCK/2u
Ygap3dzMbN40Sq7PGuBigJbV9Ek/SjqiMbAQJYgzUS6mbY9RZaXt2Q==
-----END RSA PRIVATE KEY-----`
)

func TestCA(t *testing.T) {
	ca, key, err := NewCA()
	assert.NoError(t, err)

	t.Logf("%s", EncodeCertPEM(ca))
	t.Logf("%s", EncodePrivateKeyPEM(key))
}

func TestGenCerts(t *testing.T) {
	apiserver := &CertsConfig{
		Config: certutil.Config{
			CommonName:   "system:admin",
			Organization: []string{"system:masters"},
			AltNames: certutil.AltNames{
				DNSNames: []string{
					"kube-apiserver.test1.svc",
				},
				IPs: []net.IP{
					net.ParseIP("127.0.0.1"),
				},
			},
			Usages: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
		},
	}

	caCert, err := DecodeCertPEM([]byte(caCertByte))
	assert.NoError(t, err)
	caKey, err := DecodePrivateKeyPEM([]byte(caKeyByte))
	assert.NoError(t, err)

	cert, key, err := NewCertAndKey(caCert, caKey, apiserver)
	assert.NoError(t, err)
	t.Logf("%s", EncodeCertPEM(cert))
	t.Logf("%s", EncodePrivateKeyPEM(key))
}

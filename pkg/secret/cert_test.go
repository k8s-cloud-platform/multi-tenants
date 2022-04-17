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

package cert

import (
	"net"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	certutil "k8s.io/client-go/util/cert"
	"k8s.io/client-go/util/keyutil"
)

func TestGenCerts(t *testing.T) {
	caCert, caKey, err := NewCACertAndKey()
	assert.NoError(t, err)

	notAfter := time.Now().Add(Duration365d * 10).UTC()
	apiserver := NewCertConfig(
		"system:admin",
		[]string{"system:masters"},
		certutil.AltNames{
			DNSNames: []string{
				"kube-apiserver.test1.svc",
			},
			IPs: []net.IP{
				[]byte("127.0.0.1"),
			},
		},
		&notAfter,
	)
	cert, key, err := NewCertAndKey(caCert, *caKey, apiserver)
	assert.NoError(t, err)
	t.Logf("%s", EncodeCertPEM(cert))

	keyEncoded, err := keyutil.MarshalPrivateKeyToPEM(key)
	assert.NoError(t, err)
	t.Logf("%s", keyEncoded)
}

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
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/k8s-cloud-platform/multi-tenants/pkg/secret"
)

func TestKubeconfig(t *testing.T) {
	caCert, caKey, err := secret.NewCA()
	assert.NoError(t, err)

	c, err := New("tenant-cluster", "127.0.0.1:6443", caCert, caKey)
	assert.NoError(t, err)

	config, err := clientcmd.Write(*c)
	assert.NoError(t, err)
	t.Logf("%s", config)
}

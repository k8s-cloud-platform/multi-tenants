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
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"errors"
	"fmt"
	"math/big"
	"time"

	certutil "k8s.io/client-go/util/cert"
)

// NewCA creates new certificate and private key for the certificate authority.
func NewCA(config *certutil.Config) (*x509.Certificate, *rsa.PrivateKey, error) {
	if config == nil {
		config = &certutil.Config{}
	}

	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, err
	}

	c, err := newSelfSignedCACert(config, key)
	if err != nil {
		return nil, nil, err
	}

	return c, key, nil
}

// NewCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func NewCertAndKey(caCert *x509.Certificate, caKey crypto.Signer, config *certutil.Config) (*x509.Certificate, *rsa.PrivateKey, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key %v", err)
	}

	cert, err := newSignedCert(config, key, caCert, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign certificate. %v", err)
	}

	return cert, key, nil
}

func NewPubAndKey() (*rsa.PublicKey, *rsa.PrivateKey, error) {
	key, err := NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key %v", err)
	}

	pub, ok := key.Public().(*rsa.PublicKey)
	if !ok {
		return nil, nil, errors.New("unable to convert key public to *rsa.PublicKey")
	}

	return pub, key, nil
}

// newSelfSignedCACert creates a CA certificate.
func newSelfSignedCACert(config *certutil.Config, key crypto.Signer) (*x509.Certificate, error) {
	config.CommonName = "ca"
	config.Organization = []string{"kcp"}

	now := time.Now().UTC()

	tmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Organization: config.Organization,
		},
		DNSNames:              config.AltNames.DNSNames,
		NotBefore:             now.Add(time.Minute * -5),
		NotAfter:              now.Add(time.Hour * 24 * 365 * 10), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		MaxPathLenZero:        true,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
		IsCA:                  true,
	}

	b, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, key.Public(), key)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(b)
}

// newSignedCert creates a signed certificate using the given CA certificate and key
func newSignedCert(config *certutil.Config, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer) (*x509.Certificate, error) {
	now := time.Now().UTC()

	certTmpl := x509.Certificate{
		SerialNumber: new(big.Int).SetInt64(0),
		Subject: pkix.Name{
			CommonName:   config.CommonName,
			Organization: config.Organization,
		},
		DNSNames:              config.AltNames.DNSNames,
		IPAddresses:           config.AltNames.IPs,
		NotBefore:             now.Add(time.Minute * -5),
		NotAfter:              now.Add(time.Hour * 24 * 365 * 10), // 10 years
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           config.Usages,
		BasicConstraintsValid: true,
		MaxPathLen:            0,
		IsCA:                  false,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}

	return x509.ParseCertificate(certDERBytes)
}

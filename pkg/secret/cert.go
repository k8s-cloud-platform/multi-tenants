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
	"crypto"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math"
	"math/big"
	"net"
	"time"

	"k8s.io/apimachinery/pkg/util/sets"
	certutil "k8s.io/client-go/util/cert"
)

const (
	// certificateBlockType is a possible value for pem.Block.Type.
	certificateBlockType = "CERTIFICATE"
	rsaKeySize           = 2048
	// Duration365d Certificate validity period
	Duration365d = time.Hour * 24 * 365
)

// generatePrivateKey Generate CA Private Key
func generatePrivateKey(keyType x509.PublicKeyAlgorithm) (crypto.Signer, error) {
	if keyType == x509.ECDSA {
		return ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	}

	return rsa.GenerateKey(rand.Reader, rsaKeySize)
}

// CertsConfig is a wrapper around certutil.Config extending it with PublicKeyAlgorithm.
type CertsConfig struct {
	certutil.Config
	NotAfter           *time.Time
	PublicKeyAlgorithm x509.PublicKeyAlgorithm
}

// EncodeCertPEM returns PEM-endcoded certificate data
func EncodeCertPEM(cert *x509.Certificate) []byte {
	block := pem.Block{
		Type:  certificateBlockType,
		Bytes: cert.Raw,
	}
	return pem.EncodeToMemory(&block)
}

// newCertificateAuthority creates new certificate and private key for the certificate authority
func newCertificateAuthority(config *CertsConfig) (*x509.Certificate, crypto.Signer, error) {
	key, err := generatePrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key while generating CA certificate %v", err)
	}

	cert, err := certutil.NewSelfSignedCACert(config.Config, key)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create self-signed CA certificate %v", err)
	}

	return cert, key, nil
}

// NewCACertAndKey The public and private keys of the root certificate are returned
func NewCACertAndKey() (*x509.Certificate, *crypto.Signer, error) {
	certCfg := &CertsConfig{Config: certutil.Config{
		CommonName:   "ca",
		Organization: []string{"kcp"},
	},
	}
	caCert, caKey, err := newCertificateAuthority(certCfg)
	if err != nil {
		return nil, nil, fmt.Errorf("failure while generating CA certificate and key: %v", err)
	}

	return caCert, &caKey, nil
}

// newSignedCert creates a signed certificate using the given CA certificate and key
func newSignedCert(cfg *CertsConfig, key crypto.Signer, caCert *x509.Certificate, caKey crypto.Signer, isCA bool) (*x509.Certificate, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).SetInt64(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	if len(cfg.CommonName) == 0 {
		return nil, errors.New("must specify a CommonName")
	}

	keyUsage := x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature
	if isCA {
		keyUsage |= x509.KeyUsageCertSign
	}

	removeDuplicateAltNames(&cfg.AltNames)

	notAfter := time.Now().Add(Duration365d).UTC()
	if cfg.NotAfter != nil {
		notAfter = *cfg.NotAfter
	}

	certTmpl := x509.Certificate{
		Subject: pkix.Name{
			CommonName:   cfg.CommonName,
			Organization: cfg.Organization,
		},
		DNSNames:              cfg.AltNames.DNSNames,
		IPAddresses:           cfg.AltNames.IPs,
		SerialNumber:          serial,
		NotBefore:             caCert.NotBefore,
		NotAfter:              notAfter,
		KeyUsage:              keyUsage,
		ExtKeyUsage:           cfg.Usages,
		BasicConstraintsValid: true,
		IsCA:                  isCA,
	}
	certDERBytes, err := x509.CreateCertificate(rand.Reader, &certTmpl, caCert, key.Public(), caKey)
	if err != nil {
		return nil, err
	}
	return x509.ParseCertificate(certDERBytes)
}

// removeDuplicateAltNames removes duplicate items in altNames.
func removeDuplicateAltNames(altNames *certutil.AltNames) {
	if altNames == nil {
		return
	}

	if altNames.DNSNames != nil {
		altNames.DNSNames = sets.NewString(altNames.DNSNames...).List()
	}

	ipsKeys := make(map[string]struct{})
	var ips []net.IP
	for _, one := range altNames.IPs {
		if _, ok := ipsKeys[one.String()]; !ok {
			ipsKeys[one.String()] = struct{}{}
			ips = append(ips, one)
		}
	}
	altNames.IPs = ips
}

// NewCertAndKey creates new certificate and key by passing the certificate authority certificate and key
func NewCertAndKey(caCert *x509.Certificate, caKey crypto.Signer, config *CertsConfig) (*x509.Certificate, crypto.Signer, error) {
	if len(config.Usages) == 0 {
		return nil, nil, errors.New("must specify at least one ExtKeyUsage")
	}

	key, err := generatePrivateKey(config.PublicKeyAlgorithm)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to create private key %v", err)
	}

	cert, err := newSignedCert(config, key, caCert, caKey, false)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to sign certificate. %v", err)
	}

	return cert, key, nil
}

// NewCertConfig create new CertConfig
func NewCertConfig(cn string, org []string, altNames certutil.AltNames, notAfter *time.Time) *CertsConfig {
	return &CertsConfig{
		Config: certutil.Config{
			CommonName:   cn,
			Organization: org,
			Usages:       []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
			AltNames:     altNames,
		},
		NotAfter: notAfter,
	}
}

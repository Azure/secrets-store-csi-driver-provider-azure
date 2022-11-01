package certificates

import (
	"crypto"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strings"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	certType       = "CERTIFICATE"
	privateKeyType = "PRIVATE KEY"
)

// ValidateCert checks if the certificate imported is valid for the
// given dns name
func ValidateCert(certData, dnsName string) {
	roots := x509.NewCertPool()
	// Without passing a pool, Go will use the system pool which will definitely not work.
	// By adding the certificate itself, a valid path can be built to a trusted root for self-signed certs.
	// Ref: https://stackoverflow.com/questions/63317763/check-validity-of-ssl-self-signed-certificate
	ok := roots.AppendCertsFromPEM([]byte(certData))
	Expect(ok).To(BeTrue())

	block, _ := pem.Decode([]byte(certData))
	cert, err := x509.ParseCertificate(block.Bytes)
	Expect(err).To(BeNil())

	opts := x509.VerifyOptions{
		DNSName: dnsName,
		Roots:   roots,
	}

	_, err = cert.Verify(opts)
	By(fmt.Sprintf("Ensuring certificate is valid for dns name %s", dnsName))
	Expect(err).To(BeNil())
}

// ValidateCertBundle validates the certificate, public key and private key returned by the provider match
// and are usable
func ValidateCertBundle(data, publicKey, privKey, dnsName string) {
	By(fmt.Sprintf("Ensuring certificate and private key is valid for dns name %s", dnsName))
	certPEMBlock, err := getCert([]byte(data))
	Expect(err).To(BeNil())

	keyPEMBlock, err := getPrivateKey([]byte(privKey))
	Expect(err).To(BeNil())

	certs, err := X509KeyPair(certPEMBlock, keyPEMBlock, []byte(publicKey), []byte{})
	Expect(err).To(BeNil())
	Expect(certs.Certificate).ToNot(BeNil())
	Expect(certs.PrivateKey).ToNot(BeNil())
}

func X509KeyPair(certPEMBlock, keyPEMBlock, pubKeyPEMBlock, pw []byte) (cert tls.Certificate, err error) {
	var certDERBlock *pem.Block
	for {
		certDERBlock, certPEMBlock = pem.Decode(certPEMBlock)
		if certDERBlock == nil {
			break
		}
		if certDERBlock.Type == "CERTIFICATE" {
			cert.Certificate = append(cert.Certificate, certDERBlock.Bytes)
		}
	}

	if len(cert.Certificate) == 0 {
		err = errors.New("crypto/tls: failed to parse certificate PEM data")
		return
	}
	var keyDERBlock *pem.Block
	for {
		keyDERBlock, keyPEMBlock = pem.Decode(keyPEMBlock)
		if keyDERBlock == nil {
			err = errors.New("crypto/tls: failed to parse key PEM data")
			return
		}
		if x509.IsEncryptedPEMBlock(keyDERBlock) {
			out, err2 := x509.DecryptPEMBlock(keyDERBlock, pw)
			if err2 != nil {
				err = err2
				return
			}
			keyDERBlock.Bytes = out
			break
		}
		if keyDERBlock.Type == "PRIVATE KEY" || strings.HasSuffix(keyDERBlock.Type, " PRIVATE KEY") {
			break
		}
	}

	cert.PrivateKey, err = parsePrivateKey(keyDERBlock.Bytes)
	if err != nil {
		return
	}
	pubKey, err := parsePublicKey(pubKeyPEMBlock)
	if err != nil {
		return
	}
	// We don't need to parse the public key for TLS, but we so do anyway
	// to check that it looks sane and matches the private key.
	x509Cert, err := x509.ParseCertificate(cert.Certificate[0])
	if err != nil {
		return
	}

	switch pub := x509Cert.PublicKey.(type) {
	case *rsa.PublicKey:
		privateKey, ok := cert.PrivateKey.(*rsa.PrivateKey)
		if !ok {
			err = errors.New("crypto/tls: private key type does not match public key type")
			return
		}
		if pub.N.Cmp(privateKey.N) != 0 {
			err = errors.New("crypto/tls: private key does not match public key")
			return
		}
		if !pub.Equal(pubKey) {
			err = errors.New("crypto/tls: public key does not match")
		}
	case *ecdsa.PublicKey:
		privateKey, ok := cert.PrivateKey.(*ecdsa.PrivateKey)
		if !ok {
			err = errors.New("crypto/tls: private key type does not match public key type")
			return
		}
		if pub.X.Cmp(privateKey.X) != 0 || pub.Y.Cmp(privateKey.Y) != 0 {
			err = errors.New("crypto/tls: private key does not match public key")
			return
		}
		if !pub.Equal(pubKey) {
			err = errors.New("crypto/tls: public key does not match")
		}
	default:
		err = errors.New("crypto/tls: unknown public key algorithm")
		return
	}
	return
}

// Attempt to parse the given private key DER block. OpenSSL 0.9.8 generates
// PKCS#1 private keys by default, while OpenSSL 1.0.0 generates PKCS#8 keys.
// OpenSSL ecparam generates SEC1 EC private keys for ECDSA. We try all three.
func parsePrivateKey(der []byte) (crypto.PrivateKey, error) {
	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey, *ecdsa.PrivateKey:
			return key, nil
		default:
			return nil, errors.New("crypto/tls: found unknown private key type in PKCS#8 wrapping")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		return key, nil
	}

	return nil, errors.New("crypto/tls: failed to parse private key")
}

func parsePublicKey(rawPem []byte) (crypto.PublicKey, error) {
	var err error
	pemBlock, _ := pem.Decode(rawPem)

	if key, err := x509.ParsePKIXPublicKey(pemBlock.Bytes); err == nil {
		return key, nil
	}
	if key, err := x509.ParsePKCS1PublicKey(pemBlock.Bytes); err == nil {
		return key, nil
	}

	return nil, fmt.Errorf("crypto/tls: failed to parse public key, err: %+v", err)
}

// getCert returns the certificate part of a cert
func getCert(data []byte) ([]byte, error) {
	var certs []byte
	for {
		pemBlock, rest := pem.Decode(data)
		if pemBlock == nil {
			break
		}
		if pemBlock.Type == certType {
			block := pem.EncodeToMemory(pemBlock)
			certs = append(certs, block...)
		}
		data = rest
	}
	return certs, nil
}

// getPrivateKey returns the private key part of a cert
func getPrivateKey(data []byte) ([]byte, error) {
	var der []byte
	var derKey []byte
	for {
		pemBlock, rest := pem.Decode(data)
		if pemBlock == nil {
			break
		}
		if pemBlock.Type != certType {
			der = pemBlock.Bytes
		}
		data = rest
	}

	if key, err := x509.ParsePKCS1PrivateKey(der); err == nil {
		derKey = x509.MarshalPKCS1PrivateKey(key)
	}

	if key, err := x509.ParsePKCS8PrivateKey(der); err == nil {
		switch key := key.(type) {
		case *rsa.PrivateKey:
			derKey = x509.MarshalPKCS1PrivateKey(key)
		case *ecdsa.PrivateKey:
			derKey, err = x509.MarshalECPrivateKey(key)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("unknown private key type found while getting key. Only rsa and ecdsa are supported")
		}
	}
	if key, err := x509.ParseECPrivateKey(der); err == nil {
		derKey, err = x509.MarshalECPrivateKey(key)
		if err != nil {
			return nil, err
		}
	}
	block := &pem.Block{
		Type:  privateKeyType,
		Bytes: derKey,
	}
	return pem.EncodeToMemory(block), nil
}

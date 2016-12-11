package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
)

const (
	sslCert string = "ssl_cert.pem"
	sslKey  string = "ssl_key.pem"
)

func loadSSLFiles() (cert, key string, err error) {
	ctx := logger.WithField("method", "loadSSLFiles")
	cert = path.Join(APP_DIR, sslCert)
	key = path.Join(APP_DIR, sslKey)

	// Load from env vars and write to file system.
	certData := os.Getenv("SSL_CERT")

	block, _ := pem.Decode([]byte(certData))
	if block == nil {
		return "", "", fmt.Errorf("failed to parse certificate PEM")
	}
	certPEM, errC := x509.ParseCertificate(block.Bytes)
	if errC != nil {
		return "", "", fmt.Errorf("failed to parse certificate: %s", errC)
	}

	if _, err := certPEM.Verify(x509.VerifyOptions{DNSName: "api.nutritionhabitchallenge.com"}); err != nil {
		ctx.WithError(err).Warn("Loaded SSL Certificate but it failed to verify.")
	}

	errF := ioutil.WriteFile(cert, []byte(certData), 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to writ cert to file: %s", errF)
	}
	ctx.WithField("file", cert).Info("Wrote cert to file.")

	keyData := os.Getenv("SSL_KEY")

	block, _ = pem.Decode([]byte(keyData))
	if block == nil {
		return "", "", fmt.Errorf("failed to parse key PEM")
	}
	_, errC = x509.ParsePKCS1PrivateKey(block.Bytes)
	if errC != nil {
		return "", "", fmt.Errorf("failed to parse key: %s", errC)
	}

	errF = ioutil.WriteFile(key, []byte(keyData), 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to writ key to file: %s", errF)
	}
	ctx.WithField("file", key).Info("Wrote key to file.")

	return cert, key, nil
}

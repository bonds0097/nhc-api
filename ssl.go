package main

import (
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"path"
)

const (
	sslCertFilename string = "ssl_cert.pem"
	sslKeyFilename  string = "ssl_key.pem"
)

func loadSSLFiles() (sslCertPath, sslKeyPath string, err error) {
	ctx := logger.WithField("method", "loadSSLFiles")
	sslCertPath = path.Join(APP_DIR, sslCertFilename)
	sslKeyPath = path.Join(APP_DIR, sslKeyFilename)

	// Write cert and key to file.
	sslCertData := []byte(sslCert)

	errF := ioutil.WriteFile(sslCertPath, sslCertData, 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to writ cert to file: %s", errF)
	}
	ctx.WithField("file", sslCertPath).Info("Wrote cert to file.")

	sslKeyData := []byte(sslKey)

	block, _ := pem.Decode(sslKeyData)
	if block == nil {
		return "", "", fmt.Errorf("failed to parse key PEM:\n%s", sslKey)
	}
	_, errC := x509.ParsePKCS1PrivateKey(block.Bytes)
	if errC != nil {
		return "", "", fmt.Errorf("failed to parse key: %s", errC)
	}

	errF = ioutil.WriteFile(sslKeyPath, sslKeyData, 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to writ key to file: %s", errF)
	}
	ctx.WithField("file", sslKeyPath).Info("Wrote key to file.")

	return sslCertPath, sslKeyPath, nil
}

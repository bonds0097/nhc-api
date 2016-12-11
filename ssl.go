package main

import (
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
		return "", "", fmt.Errorf("failed to write cert to file: %s", errF)
	}
	ctx.WithField("file", sslCertPath).Info("Wrote cert to file.")

	sslKeyData := []byte(sslKey)
	errF = ioutil.WriteFile(sslKeyPath, sslKeyData, 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to write key to file: %s", errF)
	}
	ctx.WithField("file", sslKeyPath).Info("Wrote key to file.")

	return sslCertPath, sslKeyPath, nil
}

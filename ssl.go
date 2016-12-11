package main

import "path"
import "os"
import "io/ioutil"
import "fmt"

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
	errF := ioutil.WriteFile(cert, []byte(certData), 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to writ cert to file: %s", errF)
	}
	ctx.WithField("file", cert).Info("Wrote cert to file.")

	keyData := os.Getenv("SSL_KEY")
	errF = ioutil.WriteFile(key, []byte(keyData), 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to writ key to file: %s", errF)
	}
	ctx.WithField("file", key).Info("Wrote key to file.")

	return cert, key, nil
}

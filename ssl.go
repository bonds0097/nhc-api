package main

import (
	"encoding/pem"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"
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
	errF := ioutil.WriteFile(sslCertPath, sslCertData, 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to write cert to file: %s", errF)
	}
	ctx.WithField("file", sslCertPath).Info("Wrote cert to file.")

	errF = ioutil.WriteFile(sslKeyPath, sslKeyData, 0644)
	if errF != nil {
		return "", "", fmt.Errorf("failed to write key to file: %s", errF)
	}
	ctx.WithField("file", sslKeyPath).Info("Wrote key to file.")

	return sslCertPath, sslKeyPath, nil
}

func loadPEMBlockFromEnv(envvar string) ([]byte, error) {
	s := os.Getenv(envvar)

	sA := strings.SplitAfter(s, "----- ")
	sA2 := strings.Split(sA[1], " -")

	sA = []string{sA[0], sA2[0], sA2[1]}
	sA3 := strings.Split(sA2[0], " ")

	sF := []string{sA[0]}
	sF = append(sF, sA3...)
	sF = append(sF, "-"+sA2[1])

	s = strings.Join(sF, "\n")

	b := []byte(s)

	block, rest := pem.Decode(b)
	if block == nil {
		return []byte{}, fmt.Errorf("failed to decode string as PEM block: %s", string(rest))
	}

	return b, nil
}

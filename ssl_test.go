package main

import "testing"

import "os"

func TestLoadPEMFromEnv(t *testing.T) {
	data := `-----BEGIN PUBLIC KEY----- MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAv/iYZP6V3ShdaibFjb4c vbmBTHiVzC80gMDSz1VQPEZrisVj0J0RCo7Or+zscjs4jU7x6ZBBlhnBYqBq1kfb zvvb5GsY7qWNgqX4ZBgLa8iInKZiIEgY5RhJz1iYhM3P5MpUFd1R2ZlYH1ZYdT/O 8aDTFmKCliDWHyxd4I8VB0ZsvHSbFXb4x5S3oiabjU6f8m9Jajdks8jdMK00CVmy d04lUM12K/nYCFk2D20IeWJvRfJaWSdMd+i/P/PrwyJrHx4IOsBnkHjyvrRMnbEc Z8UgC9LLQDRPqjqnNNTNdRxp1Zu6Atc7f4WqJerpXqlsLnlve6iqD+Wtm75QkCdP vwIDAQAB -----END PUBLIC KEY-----`

	key := "TEST_VAR"
	os.Setenv(key, data)

	_, err := loadPEMBlockFromEnv(key)
	if err != nil {
		t.Errorf("expected nil error got %s", err)
	}
}

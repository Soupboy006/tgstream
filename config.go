package main

import (
	"fmt"
	"os"
	"strconv"
)

func requiredEnvStr(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("%s is required", key)
	}
	return v, nil
}

func requiredEnvInt(key string) (int, error) {
	v, err := requiredEnvStr(key)
	if err != nil {
		return 0, err
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("%s must be a number: %w", key, err)
	}
	return n, nil
}

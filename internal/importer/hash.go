package importer

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// CalculateFileHash computes SHA-256 hash of a file
func CalculateFileHash(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("failed to hash file: %w", err)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// CalculateFileHashes computes hashes for both job and invoice files
func CalculateFileHashes(jobsPath, invoicesPath string) (jobsHash, invoicesHash string, err error) {
	jobsHash, err = CalculateFileHash(jobsPath)
	if err != nil {
		return "", "", fmt.Errorf("jobs file: %w", err)
	}

	invoicesHash, err = CalculateFileHash(invoicesPath)
	if err != nil {
		return "", "", fmt.Errorf("invoices file: %w", err)
	}

	return jobsHash, invoicesHash, nil
}

package fsutil

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// AtomicWrite writes data to a file atomically
func AtomicWrite(path string, data []byte, perm os.FileMode) error {
	// Create temporary file in same directory
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, filepath.Base(path)+".tmp")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write data to temp file
	if _, err := tmpFile.Write(data); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to write to temp file: %w", err)
	}

	// Set permissions
	if err := tmpFile.Chmod(perm); err != nil {
		tmpFile.Close()
		return fmt.Errorf("failed to set temp file permissions: %w", err)
	}

	// Close temp file
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpFile.Name(), path); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
	}

	return nil
}

// CreateBackup creates a backup of a file with timestamp
func CreateBackup(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return nil // No file to backup
	}

	timestamp := time.Now().Format("20060102150405")
	backupPath := path + ".bak." + timestamp

	// Copy file to backup
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	return nil
}

// ReplaceBetweenMarkers replaces content between start and end markers
func ReplaceBetweenMarkers(content, startMarker, endMarker, newContent string) (string, bool) {
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		// Markers not found, append new content
		if content == "" {
			return newContent, true
		}
		return content + "\n" + newContent, true
	}

	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		// Start marker found but no end marker, append
		return content + "\n" + newContent, true
	}

	endIdx += startIdx + len(endMarker)

	// Replace content between markers
	before := content[:startIdx]
	after := content[endIdx:]

	result := before + newContent + "\n" + after
	return result, true
}

// ExtractBetweenMarkers extracts content between start and end markers
func ExtractBetweenMarkers(content, startMarker, endMarker string) (string, bool) {
	startIdx := strings.Index(content, startMarker)
	if startIdx == -1 {
		return "", false
	}

	endIdx := strings.Index(content[startIdx:], endMarker)
	if endIdx == -1 {
		return "", false
	}

	startIdx += len(startMarker)
	endIdx += startIdx

	// Extract content between markers
	extracted := content[startIdx:endIdx]
	extracted = strings.TrimSpace(extracted)

	return extracted, true
}

// EnsureDir ensures a directory exists
func EnsureDir(path string) error {
	return os.MkdirAll(path, 0755)
}

// FileExists checks if a file exists
func FileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

// IsEmpty checks if a file is empty
func IsEmpty(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return true
	}
	return info.Size() == 0
}

package backup

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupService orchestrates backup operations
type BackupService struct {
	config     BackupConfig
	encryptor  *Encryptor
	uploader   Uploader
}

// BackupConfig holds backup configuration
type BackupConfig struct {
	Enabled                    bool
	FullBackupIntervalHours    int
	IncrementalIntervalMinutes int
	RetentionDays              int
	LocalPath                  string
	EncryptionKey              string
	CloudProvider              string
	CloudBucket                string
	CloudRegion                string
}

// BackupType represents the type of backup
type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
)

// BackupMetadata contains backup information
type BackupMetadata struct {
	ID          string
	Type        BackupType
	Timestamp   time.Time
	SizeBytes   int64
	Checksum    string
	Encrypted   bool
	Compressed  bool
	Duration    time.Duration
}

// BackupResult represents the result of a backup operation
type BackupResult struct {
	Success bool
	Metadata *BackupMetadata
	Error    error
}

// Uploader interface for cloud storage
type Uploader interface {
	Upload(ctx context.Context, key string, reader io.Reader) error
	Download(ctx context.Context, key string, writer io.Writer) error
	List(ctx context.Context, prefix string) ([]string, error)
	Delete(ctx context.Context, key string) error
}

// NewBackupService creates a new backup service
func NewBackupService(config BackupConfig) (*BackupService, error) {
	var encryptor *Encryptor
	if config.EncryptionKey != "" {
		e, err := NewEncryptor(config.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to create encryptor: %w", err)
		}
		encryptor = e
	}

	var uploader Uploader
	switch config.CloudProvider {
	case "s3":
		uploader = NewS3Uploader(config.CloudBucket, config.CloudRegion)
	case "gcs":
		uploader = NewGCSUploader(config.CloudBucket)
	default:
		uploader = NewLocalUploader(config.LocalPath)
	}

	return &BackupService{
		config:    config,
		encryptor: encryptor,
		uploader:  uploader,
	}, nil
}

// PerformFullBackup creates a full database backup
func (bs *BackupService) PerformFullBackup(ctx context.Context, dbPath string) (*BackupResult, error) {
	start := time.Now()
	backupID := fmt.Sprintf("full_%d", start.Unix())

	metadata := &BackupMetadata{
		ID:        backupID,
		Type:      BackupTypeFull,
		Timestamp: start,
		Encrypted: bs.encryptor != nil,
		Compressed: true,
	}

	// Create backup file
	backupFile, err := bs.createBackupFile(dbPath, backupID)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	defer os.Remove(backupFile) // Clean up local file

	// Calculate checksum
	checksum, err := bs.calculateChecksum(backupFile)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	metadata.Checksum = checksum

	// Get file size
	fileInfo, err := os.Stat(backupFile)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	metadata.SizeBytes = fileInfo.Size()

	// Upload to cloud/local
	key := fmt.Sprintf("backups/%s/%s.db.gz", metadata.Type, backupID)
	if bs.encryptor != nil {
		key += ".enc"
	}

	file, err := os.Open(backupFile)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	defer file.Close()

	if err := bs.uploader.Upload(ctx, key, file); err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}

	metadata.Duration = time.Since(start)
	return &BackupResult{Success: true, Metadata: metadata}, nil
}

// PerformIncrementalBackup creates an incremental backup
func (bs *BackupService) PerformIncrementalBackup(ctx context.Context, dbPath string) (*BackupResult, error) {
	start := time.Now()
	backupID := fmt.Sprintf("incr_%d", start.Unix())

	metadata := &BackupMetadata{
		ID:        backupID,
		Type:      BackupTypeIncremental,
		Timestamp: start,
		Encrypted: bs.encryptor != nil,
		Compressed: true,
	}

	// For incremental backup, we backup WAL files and recent changes
	// This is a simplified implementation - in production you'd track changes
	backupFile, err := bs.createIncrementalBackup(dbPath, backupID)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	defer os.Remove(backupFile)

	// Calculate checksum and size
	checksum, err := bs.calculateChecksum(backupFile)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	metadata.Checksum = checksum

	fileInfo, err := os.Stat(backupFile)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	metadata.SizeBytes = fileInfo.Size()

	// Upload
	key := fmt.Sprintf("backups/%s/%s.wal.gz", metadata.Type, backupID)
	if bs.encryptor != nil {
		key += ".enc"
	}

	file, err := os.Open(backupFile)
	if err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}
	defer file.Close()

	if err := bs.uploader.Upload(ctx, key, file); err != nil {
		return &BackupResult{Success: false, Error: err}, err
	}

	metadata.Duration = time.Since(start)
	return &BackupResult{Success: true, Metadata: metadata}, nil
}

// RestoreBackup restores from a backup
func (bs *BackupService) RestoreBackup(ctx context.Context, backupID string, targetPath string) error {
	// Find backup file
	keys, err := bs.uploader.List(ctx, "backups/")
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	var backupKey string
	for _, key := range keys {
		if strings.Contains(key, backupID) {
			backupKey = key
			break
		}
	}

	if backupKey == "" {
		return fmt.Errorf("backup %s not found", backupID)
	}

	// Download and restore
	file, err := os.CreateTemp("", "restore_*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(file.Name())
	defer file.Close()

	if err := bs.uploader.Download(ctx, backupKey, file); err != nil {
		return fmt.Errorf("failed to download backup: %w", err)
	}

	// Decrypt if needed
	if bs.encryptor != nil && strings.HasSuffix(backupKey, ".enc") {
		if err := bs.decryptFile(file.Name()); err != nil {
			return fmt.Errorf("failed to decrypt backup: %w", err)
		}
	}

	// Decompress and restore
	if err := bs.restoreFromFile(file.Name(), targetPath); err != nil {
		return fmt.Errorf("failed to restore: %w", err)
	}

	return nil
}

// CleanupOldBackups removes backups older than retention period
func (bs *BackupService) CleanupOldBackups(ctx context.Context) error {
	if bs.config.RetentionDays == 0 {
		return nil
	}

	keys, err := bs.uploader.List(ctx, "backups/")
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -bs.config.RetentionDays)

	for _, key := range keys {
		// Extract timestamp from key
		parts := strings.Split(key, "/")
		if len(parts) < 3 {
			continue
		}

		timestampStr := strings.TrimSuffix(strings.TrimSuffix(parts[2], ".db.gz"), ".wal.gz")
		if strings.HasPrefix(timestampStr, "full_") || strings.HasPrefix(timestampStr, "incr_") {
			timestampUnix := strings.TrimPrefix(timestampStr, "full_")
			timestampUnix = strings.TrimPrefix(timestampUnix, "incr_")

			if ts, err := time.Parse("1136214245", timestampUnix); err == nil {
				if ts.Before(cutoff) {
					if err := bs.uploader.Delete(ctx, key); err != nil {
						// Log error but continue
						continue
					}
				}
			}
		}
	}

	return nil
}

// Helper methods

func (bs *BackupService) createBackupFile(dbPath, backupID string) (string, error) {
	// Simplified - in production you'd copy the database file
	// while holding a read lock
	tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("backup_%s.db", backupID))

	// Copy database file
	sourceFile, err := os.Open(dbPath)
	if err != nil {
		return "", err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(tempFile)
	if err != nil {
		return "", err
	}
	defer destFile.Close()

	if _, err := io.Copy(destFile, sourceFile); err != nil {
		return "", err
	}

	// Compress
	compressedFile := tempFile + ".gz"
	if err := compressFile(tempFile, compressedFile); err != nil {
		return "", err
	}

	// Encrypt if configured
	if bs.encryptor != nil {
		encryptedFile := compressedFile + ".enc"
		if err := bs.encryptor.EncryptFile(compressedFile, encryptedFile); err != nil {
			return "", err
		}
		os.Remove(compressedFile)
		return encryptedFile, nil
	}

	os.Remove(tempFile)
	return compressedFile, nil
}

func (bs *BackupService) createIncrementalBackup(dbPath, backupID string) (string, error) {
	// Simplified - backup WAL file
	walPath := dbPath + "-wal"
	if _, err := os.Stat(walPath); os.IsNotExist(err) {
		// No WAL file, create empty backup
		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("incr_%s.wal", backupID))
		err := os.WriteFile(tempFile, []byte(""), 0644)
		return tempFile, err
	}

	return bs.createBackupFile(walPath, backupID)
}

func (bs *BackupService) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func (bs *BackupService) decryptFile(filePath string) error {
	if bs.encryptor == nil {
		return nil
	}
	return bs.encryptor.DecryptFile(filePath, filePath+".dec")
}

func (bs *BackupService) restoreFromFile(backupPath, targetPath string) error {
	// Simplified restore - in production you'd handle decompression/decryption
	return copyFile(backupPath, targetPath)
}

// Utility functions
func compressFile(src, dst string) error {
	// Simplified - use gzip in production
	return copyFile(src, dst)
}

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)
	return err
}
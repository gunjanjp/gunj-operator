package storage

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/go-logr/logr"
	"github.com/gunjanjp/gunj-operator/internal/backup"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Manager handles storage operations for backups
type Manager struct {
	log    logr.Logger
	client client.Client
}

// NewManager creates a new storage manager
func NewManager(log logr.Logger) (*Manager, error) {
	return &Manager{
		log: log.WithName("storage-manager"),
	}, nil
}

// SetClient sets the Kubernetes client
func (m *Manager) SetClient(client client.Client) {
	m.client = client
}

// CreateBackupArchive creates a tar.gz archive from backup items
func (m *Manager) CreateBackupArchive(items []backup.BackupItem) ([]byte, error) {
	m.log.V(1).Info("Creating backup archive", "items", len(items))
	
	// Create buffer for archive
	var buf bytes.Buffer
	
	// Create gzip writer
	gzipWriter := gzip.NewWriter(&buf)
	defer gzipWriter.Close()
	
	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()
	
	// Create manifest
	manifest := BackupManifest{
		Version:    "1.0",
		ItemsCount: len(items),
		Items:      make([]ManifestItem, 0, len(items)),
	}
	
	// Add each item to archive
	for i, item := range items {
		// Serialize item data
		data, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("marshaling item %s: %w", item.Name, err)
		}
		
		// Create tar header
		header := &tar.Header{
			Name: fmt.Sprintf("items/%d_%s_%s.json", i, item.GroupVersionKind.Kind, item.Name),
			Mode: 0644,
			Size: int64(len(data)),
		}
		
		// Write header
		if err := tarWriter.WriteHeader(header); err != nil {
			return nil, fmt.Errorf("writing tar header: %w", err)
		}
		
		// Write data
		if _, err := tarWriter.Write(data); err != nil {
			return nil, fmt.Errorf("writing tar data: %w", err)
		}
		
		// Add to manifest
		manifest.Items = append(manifest.Items, ManifestItem{
			Name:             item.Name,
			Namespace:        item.Namespace,
			GroupVersionKind: item.GroupVersionKind,
			Checksum:         m.calculateChecksum(data),
		})
	}
	
	// Add manifest to archive
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshaling manifest: %w", err)
	}
	
	manifestHeader := &tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(manifestData)),
	}
	
	if err := tarWriter.WriteHeader(manifestHeader); err != nil {
		return nil, fmt.Errorf("writing manifest header: %w", err)
	}
	
	if _, err := tarWriter.Write(manifestData); err != nil {
		return nil, fmt.Errorf("writing manifest data: %w", err)
	}
	
	// Close writers to flush data
	if err := tarWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing tar writer: %w", err)
	}
	
	if err := gzipWriter.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}
	
	archiveData := buf.Bytes()
	m.log.Info("Created backup archive", "size", len(archiveData))
	
	return archiveData, nil
}

// ExtractBackupArchive extracts backup items from a tar.gz archive
func (m *Manager) ExtractBackupArchive(data []byte) ([]backup.BackupItem, error) {
	m.log.V(1).Info("Extracting backup archive", "size", len(data))
	
	// Create reader
	buf := bytes.NewReader(data)
	
	// Create gzip reader
	gzipReader, err := gzip.NewReader(buf)
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer gzipReader.Close()
	
	// Create tar reader
	tarReader := tar.NewReader(gzipReader)
	
	var items []backup.BackupItem
	var manifest *BackupManifest
	
	// Read all files from archive
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("reading tar header: %w", err)
		}
		
		// Read file data
		fileData := make([]byte, header.Size)
		if _, err := io.ReadFull(tarReader, fileData); err != nil {
			return nil, fmt.Errorf("reading file data: %w", err)
		}
		
		// Handle manifest
		if header.Name == "manifest.json" {
			manifest = &BackupManifest{}
			if err := json.Unmarshal(fileData, manifest); err != nil {
				return nil, fmt.Errorf("unmarshaling manifest: %w", err)
			}
			continue
		}
		
		// Handle backup items
		if header.Name[:6] == "items/" {
			var item backup.BackupItem
			if err := json.Unmarshal(fileData, &item); err != nil {
				return nil, fmt.Errorf("unmarshaling item from %s: %w", header.Name, err)
			}
			items = append(items, item)
		}
	}
	
	// Validate against manifest if present
	if manifest != nil {
		if len(items) != manifest.ItemsCount {
			return nil, fmt.Errorf("item count mismatch: expected %d, got %d", manifest.ItemsCount, len(items))
		}
	}
	
	m.log.Info("Extracted backup archive", "items", len(items))
	
	return items, nil
}

// CompressData compresses data using the specified algorithm
func (m *Manager) CompressData(data []byte, config *backup.CompressionConfig) ([]byte, error) {
	switch config.Algorithm {
	case "gzip":
		return m.compressGzip(data, config.Level)
	case "zstd":
		return m.compressZstd(data, config.Level)
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", config.Algorithm)
	}
}

// DecompressData decompresses data
func (m *Manager) DecompressData(data []byte, algorithm string) ([]byte, error) {
	switch algorithm {
	case "gzip":
		return m.decompressGzip(data)
	case "zstd":
		return m.decompressZstd(data)
	default:
		return nil, fmt.Errorf("unsupported compression algorithm: %s", algorithm)
	}
}

// compressGzip compresses data using gzip
func (m *Manager) compressGzip(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	
	// Create gzip writer with compression level
	var writer *gzip.Writer
	var err error
	
	switch level {
	case 0:
		writer, err = gzip.NewWriterLevel(&buf, gzip.NoCompression)
	case 1:
		writer, err = gzip.NewWriterLevel(&buf, gzip.BestSpeed)
	case 9:
		writer, err = gzip.NewWriterLevel(&buf, gzip.BestCompression)
	default:
		writer, err = gzip.NewWriterLevel(&buf, gzip.DefaultCompression)
	}
	
	if err != nil {
		return nil, fmt.Errorf("creating gzip writer: %w", err)
	}
	defer writer.Close()
	
	// Write data
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("writing compressed data: %w", err)
	}
	
	// Close to flush
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("closing gzip writer: %w", err)
	}
	
	return buf.Bytes(), nil
}

// decompressGzip decompresses gzip data
func (m *Manager) decompressGzip(data []byte) ([]byte, error) {
	reader, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("creating gzip reader: %w", err)
	}
	defer reader.Close()
	
	decompressed, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("reading decompressed data: %w", err)
	}
	
	return decompressed, nil
}

// compressZstd compresses data using zstandard
func (m *Manager) compressZstd(data []byte, level int) ([]byte, error) {
	// This would require the zstd library
	// For now, we'll return an error
	return nil, fmt.Errorf("zstd compression not implemented")
}

// decompressZstd decompresses zstandard data
func (m *Manager) decompressZstd(data []byte) ([]byte, error) {
	// This would require the zstd library
	// For now, we'll return an error
	return nil, fmt.Errorf("zstd decompression not implemented")
}

// EncryptData encrypts data using AES-256-GCM
func (m *Manager) EncryptData(ctx context.Context, data []byte, config *backup.EncryptionConfig) ([]byte, error) {
	// Get encryption key from secret
	key, err := m.getEncryptionKey(ctx, config.KeyRef)
	if err != nil {
		return nil, fmt.Errorf("getting encryption key: %w", err)
	}
	
	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	
	// Create nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generating nonce: %w", err)
	}
	
	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	
	// Create encrypted data structure
	encrypted := EncryptedData{
		Algorithm:  config.Algorithm,
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext[gcm.NonceSize():]),
	}
	
	// Marshal to JSON
	encryptedData, err := json.Marshal(encrypted)
	if err != nil {
		return nil, fmt.Errorf("marshaling encrypted data: %w", err)
	}
	
	return encryptedData, nil
}

// DecryptData decrypts data
func (m *Manager) DecryptData(ctx context.Context, data []byte, keyRef *corev1.SecretKeySelector) ([]byte, error) {
	// Parse encrypted data
	var encrypted EncryptedData
	if err := json.Unmarshal(data, &encrypted); err != nil {
		return nil, fmt.Errorf("unmarshaling encrypted data: %w", err)
	}
	
	// Get decryption key
	key, err := m.getEncryptionKey(ctx, keyRef)
	if err != nil {
		return nil, fmt.Errorf("getting decryption key: %w", err)
	}
	
	// Decode nonce and ciphertext
	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, fmt.Errorf("decoding nonce: %w", err)
	}
	
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, fmt.Errorf("decoding ciphertext: %w", err)
	}
	
	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}
	
	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}
	
	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypting data: %w", err)
	}
	
	return plaintext, nil
}

// getEncryptionKey retrieves encryption key from secret
func (m *Manager) getEncryptionKey(ctx context.Context, keyRef *corev1.SecretKeySelector) ([]byte, error) {
	if m.client == nil {
		return nil, fmt.Errorf("kubernetes client not set")
	}
	
	// Get secret
	secret := &corev1.Secret{}
	if err := m.client.Get(ctx, client.ObjectKey{
		Name:      keyRef.Name,
		Namespace: keyRef.Namespace,
	}, secret); err != nil {
		return nil, fmt.Errorf("getting secret: %w", err)
	}
	
	// Get key from secret
	keyData, exists := secret.Data[keyRef.Key]
	if !exists {
		return nil, fmt.Errorf("key %s not found in secret", keyRef.Key)
	}
	
	// Ensure key is 32 bytes for AES-256
	if len(keyData) != 32 {
		// Hash the key to get 32 bytes
		hash := sha256.Sum256(keyData)
		keyData = hash[:]
	}
	
	return keyData, nil
}

// calculateChecksum calculates SHA256 checksum
func (m *Manager) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

// ValidateChecksum validates data checksum
func (m *Manager) ValidateChecksum(data []byte, expectedChecksum string) bool {
	actualChecksum := m.calculateChecksum(data)
	return actualChecksum == expectedChecksum
}

// BackupManifest represents the backup manifest
type BackupManifest struct {
	Version    string         `json:"version"`
	ItemsCount int            `json:"itemsCount"`
	Items      []ManifestItem `json:"items"`
	Timestamp  string         `json:"timestamp,omitempty"`
	Checksum   string         `json:"checksum,omitempty"`
}

// ManifestItem represents an item in the manifest
type ManifestItem struct {
	Name             string                 `json:"name"`
	Namespace        string                 `json:"namespace,omitempty"`
	GroupVersionKind map[string]interface{} `json:"groupVersionKind"`
	Checksum         string                 `json:"checksum"`
}

// EncryptedData represents encrypted backup data
type EncryptedData struct {
	Algorithm  string `json:"algorithm"`
	Nonce      string `json:"nonce"`
	Ciphertext string `json:"ciphertext"`
	KeyID      string `json:"keyId,omitempty"`
}

// IsCompressed checks if data is compressed
func IsCompressed(data []byte) bool {
	// Check for gzip magic number
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return true
	}
	
	// Check for zstd magic number
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
		return true
	}
	
	return false
}

// IsEncrypted checks if data is encrypted
func IsEncrypted(data []byte) bool {
	// Try to parse as encrypted data
	var encrypted EncryptedData
	err := json.Unmarshal(data, &encrypted)
	return err == nil && encrypted.Algorithm != "" && encrypted.Ciphertext != ""
}

// GetCompressionAlgorithm detects compression algorithm
func GetCompressionAlgorithm(data []byte) string {
	// Check for gzip
	if len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b {
		return "gzip"
	}
	
	// Check for zstd
	if len(data) >= 4 && data[0] == 0x28 && data[1] == 0xb5 && data[2] == 0x2f && data[3] == 0xfd {
		return "zstd"
	}
	
	return ""
}

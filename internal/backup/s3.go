package backup

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Config — параметры S3-резервного копирования.
type Config struct {
	Endpoint    string
	Bucket      string
	AccessKey   string
	SecretKey   string
	EncKeyHex   string // AES-256-GCM (64 hex-символа, 32 байта)
	UseTLS      bool
	Retention   int
}

// Enabled — все обязательные поля заполнены.
func (c Config) Enabled() bool {
	return c.Endpoint != "" && c.Bucket != "" && c.AccessKey != "" &&
		c.SecretKey != "" && c.EncKeyHex != ""
}

// EncryptAESGCM шифрует данные ключом в hex-формате (32 байта).
// Формат блоба: nonce(12) || ciphertext || tag(16).
func EncryptAESGCM(keyHex string, plain []byte) ([]byte, error) {
	keyHex = strings.ReplaceAll(strings.TrimSpace(keyHex), " ", "")
	if len(keyHex) != 64 {
		return nil, fmt.Errorf("S3_ENC_KEY must be 64 hex chars (32 bytes), got %d", len(keyHex))
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		if _, err := fmt.Sscanf(keyHex[i*2:i*2+2], "%02x", &key[i]); err != nil {
			return nil, fmt.Errorf("invalid hex in S3_ENC_KEY: %v", err)
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return gcm.Seal(nonce, nonce, plain, nil), nil
}

// DecryptAESGCM расшифровывает блоб, созданный EncryptAESGCM.
func DecryptAESGCM(keyHex string, blob []byte) ([]byte, error) {
	keyHex = strings.ReplaceAll(strings.TrimSpace(keyHex), " ", "")
	if len(keyHex) != 64 {
		return nil, fmt.Errorf("key must be 64 hex chars")
	}
	key := make([]byte, 32)
	for i := 0; i < 32; i++ {
		if _, err := fmt.Sscanf(keyHex[i*2:i*2+2], "%02x", &key[i]); err != nil {
			return nil, err
		}
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	ns := gcm.NonceSize()
	if len(blob) < ns {
		return nil, fmt.Errorf("blob too short")
	}
	return gcm.Open(nil, blob[:ns], blob[ns:], nil)
}

func newClient(cfg Config) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseTLS,
	})
}

// UploadFile шифрует файл и загружает его в S3/MinIO, затем ротирует старые.
func UploadFile(cfg Config, localPath string) (string, error) {
	plain, err := os.ReadFile(localPath)
	if err != nil {
		return "", fmt.Errorf("read: %w", err)
	}
	encrypted, err := EncryptAESGCM(cfg.EncKeyHex, plain)
	if err != nil {
		return "", err
	}
	client, err := newClient(cfg)
	if err != nil {
		return "", fmt.Errorf("minio: %w", err)
	}
	ctx := context.Background()
	exists, err := client.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return "", fmt.Errorf("bucket check: %w", err)
	}
	if !exists {
		if err := client.MakeBucket(ctx, cfg.Bucket, minio.MakeBucketOptions{}); err != nil {
			return "", fmt.Errorf("make bucket: %w", err)
		}
	}
	objName := "montazhnik/db-backup-" + time.Now().Format("20060102-150405") + ".enc"
	_, err = client.PutObject(ctx, cfg.Bucket, objName,
		strings.NewReader(string(encrypted)), int64(len(encrypted)), minio.PutObjectOptions{})
	if err != nil {
		return "", fmt.Errorf("put: %w", err)
	}
	log.Printf("S3 backup uploaded: %s (%d bytes encrypted)", objName, len(encrypted))
	rotate(client, cfg)
	return objName, nil
}

func rotate(client *minio.Client, cfg Config) {
	retention := cfg.Retention
	if retention <= 0 {
		retention = 7
	}
	ctx := context.Background()
	const prefix = "montazhnik/db-backup-"
	var all []minio.ObjectInfo
	objs := client.ListObjects(ctx, cfg.Bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: true})
	for o := range objs {
		if o.Err != nil {
			log.Printf("S3 list: %v", o.Err)
			continue
		}
		all = append(all, o)
	}
	sort.Slice(all, func(i, j int) bool { return all[i].LastModified.Before(all[j].LastModified) })
	for i := 0; i < len(all)-retention; i++ {
		if err := client.RemoveObject(ctx, cfg.Bucket, all[i].Key, minio.RemoveObjectOptions{}); err != nil {
			log.Printf("S3 remove %s: %v", all[i].Key, err)
		} else {
			log.Printf("S3 rotated out: %s", all[i].Key)
		}
	}
}

// ListBackupFiles — файлы дампов в каталоге.
func ListBackupFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".dump") || strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// LatestFile — самый свежий по времени модификации файл.
func LatestFile(files []string) string {
	latest := ""
	var latestTime time.Time
	for _, f := range files {
		if st, err := os.Stat(f); err == nil && st.ModTime().After(latestTime) {
			latest = f
			latestTime = st.ModTime()
		}
	}
	return latest
}
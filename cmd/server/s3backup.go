package main

import (
	"log"

	"github.com/yourorg/montazhnik/backend/internal/backup"
)

// s3BackupConfig собирает конфигурацию S3 из Config приложения.
func s3BackupConfig(cfg Config) backup.Config {
	return backup.Config{
		Endpoint:  cfg.S3Endpoint,
		Bucket:    cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		EncKeyHex: cfg.S3EncKeyHex,
		UseTLS:    cfg.S3UseTLS,
		Retention: 7,
	}
}

// doBackupAndS3 выполняет локальный pg_dump, затем (если S3 настроен) загружает
// самый свежий дамп в S3/MinIO с клиентским шифрованием.
func doBackupAndS3(cfg Config) {
	backupDB(cfg.DatabaseURL)

	s3cfg := s3BackupConfig(cfg)
	if !s3cfg.Enabled() {
		return
	}
	files, err := backup.ListBackupFiles("C:/1.2 MS/postgres/backups")
	if err != nil {
		log.Printf("S3 backup: list failed: %v", err)
		return
	}
	latest := backup.LatestFile(files)
	if latest == "" {
		log.Printf("S3 backup: no dumps to upload")
		return
	}
	if _, err := backup.UploadFile(s3cfg, latest); err != nil {
		log.Printf("S3 backup failed: %v", err)
	}
}
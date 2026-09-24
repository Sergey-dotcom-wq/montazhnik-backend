package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/yourorg/montazhnik/backend/internal/backup"
)

func ctxBg() context.Context { return context.Background() }

// CLI-проверка S3-бэкапа:
//   go run ./cmd/s3check upload <file>   — зашифровать и загрузить файл, затем скачать обратно и расшифровать
//   go run ./cmd/s3check list            — список объектов в бакете
//
// Параметры берутся из окружения: S3_ENDPOINT, S3_BUCKET, S3_ACCESS_KEY,
// S3_SECRET_KEY, S3_ENC_KEY, S3_USE_TLS.
func main() {
	cfg := backup.Config{
		Endpoint:  os.Getenv("S3_ENDPOINT"),
		Bucket:    os.Getenv("S3_BUCKET"),
		AccessKey: os.Getenv("S3_ACCESS_KEY"),
		SecretKey: os.Getenv("S3_SECRET_KEY"),
		EncKeyHex: os.Getenv("S3_ENC_KEY"),
		UseTLS:    os.Getenv("S3_USE_TLS") == "1" || os.Getenv("S3_USE_TLS") == "true",
		Retention: 7,
	}
	if !cfg.Enabled() {
		log.Fatal("S3 not configured: set S3_ENDPOINT/BUCKET/ACCESS_KEY/SECRET_KEY/ENC_KEY")
	}

	cmd := "upload"
	arg := ""
	if len(os.Args) > 1 {
		cmd = os.Args[1]
	}
	if len(os.Args) > 2 {
		arg = os.Args[2]
	}

	switch cmd {
	case "upload":
		if arg == "" {
			log.Fatal("usage: s3check upload <file>")
		}
		obj, err := backup.UploadFile(cfg, arg)
		if err != nil {
			log.Fatalf("upload failed: %v", err)
		}
		fmt.Println("uploaded:", obj)

		// Скачиваем обратно и расшифровываем — проверка сквозного цикла.
		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseTLS,
		})
		if err != nil {
			log.Fatal(err)
		}
		obj2, err := client.GetObject(ctxBg(), cfg.Bucket, obj, minio.GetObjectOptions{})
		if err != nil {
			log.Fatalf("get failed: %v", err)
		}
		var encBuf []byte
		buf := make([]byte, 4096)
		for {
			n, rerr := obj2.Read(buf)
			if n > 0 {
				encBuf = append(encBuf, buf[:n]...)
			}
			if rerr != nil {
				break
			}
		}
		obj2.Close()
		plain, err := backup.DecryptAESGCM(cfg.EncKeyHex, encBuf)
		if err != nil {
			log.Fatalf("decrypt failed: %v", err)
		}
		fmt.Printf("roundtrip OK: encrypted %d -> decrypted %d bytes\n", len(encBuf), len(plain))
		fmt.Println("decrypted head:", strings.SplitN(string(plain), "\n", 2)[0])

	case "list":
		client, err := minio.New(cfg.Endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
			Secure: cfg.UseTLS,
		})
		if err != nil {
			log.Fatal(err)
		}
		for o := range client.ListObjects(ctxBg(), cfg.Bucket, minio.ListObjectsOptions{Prefix: "montazhnik/", Recursive: true}) {
			if o.Err != nil {
				log.Printf("err: %v", o.Err)
				continue
			}
			fmt.Printf("%-50s %8d bytes  %s\n", o.Key, o.Size, o.LastModified.Format("2006-01-02 15:04:05"))
		}
	default:
		log.Fatalf("unknown command: %s", cmd)
	}
}
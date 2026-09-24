package main

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"time"
)

// Генератор PKI для mTLS (инфраструктуры сертификатов):
//   1. корневой CA (ca.key + ca.crt)          — приватный ключ хранить у владельца
//   2. серверный сертификат (server.key + server.crt) с SANs локальной сети
//   3. клиентский сертификат устройства (client.key + client.crt) — ExtKeyUsage ClientAuth
//
// Использование:
//   go run ./cmd/gen_mtls   → создаёт ca/servern/client сертификаты в текущем каталоге.
// Повторный запуск НЕ перезаписывает существующий CA (только до-выпускает клиентов).

const rsaBits = 2048

type keypair struct {
	key *rsa.PrivateKey
}

func genKey() *rsa.PrivateKey {
	k, err := rsa.GenerateKey(rand.Reader, rsaBits)
	if err != nil {
		panic(err)
	}
	return k
}

func writePEM(path, blockType string, data []byte) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	if err := pem.Encode(f, &pem.Block{Type: blockType, Bytes: data}); err != nil {
		panic(err)
	}
	println("created:", path)
}

func writeKey(path string, k *rsa.PrivateKey) {
	writePEM(path, "RSA PRIVATE KEY", x509.MarshalPKCS1PrivateKey(k))
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func main() {
	// ===== 1. Корневой CA =====
	if fileExists("ca.crt") && fileExists("ca.key") {
		println("CA already exists, reusing ca.crt / ca.key")
	} else {
		caKey := genKey()
		caTemplate := &x509.Certificate{
			SerialNumber:          big.NewInt(time.Now().UnixNano()),
			Subject:               pkix.Name{CommonName: "Montazhnik Svyazi Root CA", Organization: []string{"Montazhnik Svyazi"}},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().AddDate(10, 0, 0), // 10 лет
			IsCA:                  true,
			KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign | x509.KeyUsageDigitalSignature,
			BasicConstraintsValid: true,
		}
		caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
		if err != nil {
			panic(err)
		}
		writeKey("ca.key", caKey)
		writePEM("ca.crt", "CERTIFICATE", caDER)
	}

	// Загружаем существующий CA
	caPEM, _ := os.ReadFile("ca.crt")
	caKeyPEM, _ := os.ReadFile("ca.key")
	caBlock, _ := pem.Decode(caPEM)
	caKeyBlock, _ := pem.Decode(caKeyPEM)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		panic(err)
	}
	caKey, err := x509.ParsePKCS1PrivateKey(caKeyBlock.Bytes)
	if err != nil {
		panic(err)
	}

	sign := func(cn string, dns []string, ips []net.IP, eku []x509.ExtKeyUsage, path string) {
		k := genKey()
		tpl := &x509.Certificate{
			SerialNumber:          big.NewInt(time.Now().UnixNano()),
			Subject:               pkix.Name{CommonName: cn, Organization: []string{"Montazhnik Svyazi"}},
			NotBefore:             time.Now().Add(-time.Hour),
			NotAfter:              time.Now().AddDate(1, 0, 0),
			KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
			ExtKeyUsage:           eku,
			BasicConstraintsValid: true,
			DNSNames:              dns,
			IPAddresses:           ips,
		}
		der, err := x509.CreateCertificate(rand.Reader, tpl, caCert, &k.PublicKey, caKey)
		if err != nil {
			panic(err)
		}
		writeKey(path+".key", k)
		writePEM(path+".crt", "CERTIFICATE", der)
	}

	// ===== 2. Серверный сертификат (если ещё нет) =====
	if !fileExists("server.crt") || !fileExists("server.key") {
		sign(
			"montazhnik-svyazi.local",
			[]string{"localhost", "montazhnik-svyazi.local"},
			[]net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("192.168.0.10"), net.ParseIP("192.168.0.11")},
			[]x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
			"server",
		)
	}

	// ===== 3. Клиентский сертификат устройства (тестер) =====
	// Выпускает сертификат для мобильного устройства: ExtKeyUsage ClientAuth.
	// Имя файла client1 — можно выпустить клиентов под каждое устройство:
	//   go run ./cmd/gen_mtls   (созадёт client1 при первом запуске)
	if !fileExists("client1.crt") || !fileExists("client1.key") {
		sign(
			"montazhnik-device",
			[]string{"device.montazhnik.local"},
			nil,
			[]x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
			"client1",
		)
	}

	// Сохраняем клиента в общий PKCS12/p12 ПРИМЕР: на Android удобнее импортировать
	// client1.p12. Для простоты выводим PEM; при необходимости конвертировать в p12
	// можно openssl: openssl pkcs12 -export -inkey client1.key -in client1.crt -certfile ca.crt -out client1.p12
	if !fileExists("client1.p12") {
		println("INFO: for Android convert to PKCS12:")
		println("  openssl pkcs12 -export -inkey client1.key -in client1.crt -certfile ca.crt -out client1.p12")
	}

	println("OK: PKI готов — ca.crt/ca.key, server.crt/server.key, client1.crt/client1.key")
}
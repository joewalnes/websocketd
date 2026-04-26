package integration

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"math/big"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// Tests for #413: --sslca for mutual TLS (client certificate verification).

func TestIssue413_MutualTLS(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	caCertFile, caKeyFile := generateCA(t, dir)
	serverCertFile, serverKeyFile := generateSignedCert(t, dir, "server", caCertFile, caKeyFile)
	clientCertFile, clientKeyFile := generateSignedCert(t, dir, "client", caCertFile, caKeyFile)

	port := freePort(t)
	cmd := exec.Command(websocketdBin,
		"--port="+strconv.Itoa(port),
		"--address=127.0.0.1",
		"--loglevel=error",
		"--ssl",
		"--sslcert="+serverCertFile,
		"--sslkey="+serverKeyFile,
		"--sslca="+caCertFile,
		testcmdBin, "echo",
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	waitForPort(t, port, 10*time.Second)

	// Load CA for client verification
	caCert, _ := os.ReadFile(caCertFile)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Load client certificate
	clientCert, err := tls.LoadX509KeyPair(clientCertFile, clientKeyFile)
	if err != nil {
		t.Fatalf("failed to load client cert: %v", err)
	}

	// Connect with valid client cert — should succeed
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs:      caCertPool,
			Certificates: []tls.Certificate{clientCert},
		},
	}
	conn, _, err := dialer.Dial("wss://127.0.0.1:"+strconv.Itoa(port)+"/", nil)
	if err != nil {
		t.Fatalf("mutual TLS with valid client cert should succeed: %v", err)
	}
	defer conn.Close()

	// Verify echo works
	conn.WriteMessage(websocket.TextMessage, []byte("mtls hello"))
	conn.SetReadDeadline(time.Now().Add(5 * time.Second))
	_, msg, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(msg) != "mtls hello" {
		t.Errorf("expected 'mtls hello', got %q", string(msg))
	}
}

func TestIssue413_MutualTLSRejectsNoClientCert(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	caCertFile, caKeyFile := generateCA(t, dir)
	serverCertFile, serverKeyFile := generateSignedCert(t, dir, "server", caCertFile, caKeyFile)

	port := freePort(t)
	cmd := exec.Command(websocketdBin,
		"--port="+strconv.Itoa(port),
		"--address=127.0.0.1",
		"--loglevel=error",
		"--ssl",
		"--sslcert="+serverCertFile,
		"--sslkey="+serverKeyFile,
		"--sslca="+caCertFile,
		testcmdBin, "echo",
	)
	if err := cmd.Start(); err != nil {
		t.Fatalf("failed to start: %v", err)
	}
	t.Cleanup(func() {
		cmd.Process.Kill()
		cmd.Wait()
	})
	waitForPort(t, port, 10*time.Second)

	caCert, _ := os.ReadFile(caCertFile)
	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Connect WITHOUT client cert — should be rejected
	dialer := websocket.Dialer{
		HandshakeTimeout: 5 * time.Second,
		TLSClientConfig: &tls.Config{
			RootCAs: caCertPool,
		},
	}
	_, _, err := dialer.Dial("wss://127.0.0.1:"+strconv.Itoa(port)+"/", nil)
	if err == nil {
		t.Fatal("connection without client cert should be rejected with --sslca")
	}
}

// generateCA creates a self-signed CA certificate and key.
func generateCA(t *testing.T, dir string) (certFile, keyFile string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}

	certFile = filepath.Join(dir, "ca-cert.pem")
	keyFile = filepath.Join(dir, "ca-key.pem")
	writePEM(t, certFile, "CERTIFICATE", certDER)
	keyDER, _ := x509.MarshalECPrivateKey(key)
	writePEM(t, keyFile, "EC PRIVATE KEY", keyDER)
	return
}

// generateSignedCert creates a certificate signed by the given CA.
func generateSignedCert(t *testing.T, dir, name, caCertFile, caKeyFile string) (certFile, keyFile string) {
	t.Helper()

	// Load CA
	caCertPEM, _ := os.ReadFile(caCertFile)
	caBlock, _ := pem.Decode(caCertPEM)
	caCert, err := x509.ParseCertificate(caBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	caKeyPEM, _ := os.ReadFile(caKeyFile)
	caKeyBlock, _ := pem.Decode(caKeyPEM)
	caKey, err := x509.ParseECPrivateKey(caKeyBlock.Bytes)
	if err != nil {
		t.Fatal(err)
	}

	// Generate new key
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	template := x509.Certificate{
		SerialNumber: big.NewInt(2),
		NotBefore:    time.Now(),
		NotAfter:     time.Now().Add(time.Hour),
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth, x509.ExtKeyUsageClientAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, caCert, &key.PublicKey, caKey)
	if err != nil {
		t.Fatal(err)
	}

	certFile = filepath.Join(dir, name+"-cert.pem")
	keyFile = filepath.Join(dir, name+"-key.pem")
	writePEM(t, certFile, "CERTIFICATE", certDER)
	keyDER, _ := x509.MarshalECPrivateKey(key)
	writePEM(t, keyFile, "EC PRIVATE KEY", keyDER)
	return
}

func writePEM(t *testing.T, path, pemType string, data []byte) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	pem.Encode(f, &pem.Block{Type: pemType, Bytes: data})
	f.Close()
}

// SPDX-License-Identifier: Apache-2.0

package zftp_test

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"math/big"
	"net"
	"testing"
	"time"

	zftp "gopkg.in/ro-ag/zftp.v2"
	"gopkg.in/ro-ag/zftp.v2/internal/mockzos"
)

// newSelfSignedTLS mints a fresh self-signed ECDSA certificate valid for the
// loopback address and returns a server config that presents it plus a client
// config that trusts it (ServerName 127.0.0.1, matching the mock's listen
// address). It lets the AUTH TLS path be exercised end-to-end with no external
// PKI.
func newSelfSignedTLS(t *testing.T) (serverCfg, clientCfg *tls.Config) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "mockzos"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(24 * time.Hour),
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IsCA:                  true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback},
		DNSNames:              []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &tmpl, &tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatalf("parse certificate: %v", err)
	}
	pool := x509.NewCertPool()
	pool.AddCert(cert)
	serverCfg = &tls.Config{
		Certificates: []tls.Certificate{{Certificate: [][]byte{der}, PrivateKey: key, Leaf: cert}},
	}
	clientCfg = &tls.Config{RootCAs: pool, ServerName: "127.0.0.1"}
	return serverCfg, clientCfg
}

// TestAuthTLS_LoginAndCommandOverTLS drives the full AUTH TLS handshake against
// the mock: AUTH TLS (234) upgrades the control connection to TLS, PBSZ/PROT
// (200) follow, then a complete login and a NOOP all run encrypted. It proves the
// conn/reader swap in AuthTLS keeps the control stream usable after the upgrade.
func TestAuthTLS_LoginAndCommandOverTLS(t *testing.T) {
	srv := mockzos.New(t)
	serverCfg, clientCfg := newSelfSignedTLS(t)
	srv.EnableTLS(serverCfg)

	s, err := zftp.Open(srv.Addr())
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { _ = s.Close() })

	if err := s.AuthTLS(clientCfg); err != nil {
		t.Fatalf("AuthTLS: %v", err)
	}
	if err := s.Login("ME", "PW"); err != nil {
		t.Fatalf("Login over TLS: %v", err)
	}
	if _, err := s.SendCommand(zftp.CodeCmdOK, "NOOP"); err != nil {
		t.Fatalf("NOOP over TLS: %v", err)
	}

	if !hasCmd(srv.Commands(), "AUTH TLS") {
		t.Errorf("AUTH TLS was not received by the server; commands=%v", srv.Commands())
	}
}

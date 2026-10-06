package tests

import (
	"bytes"
	"crypto/x509"
	"encoding/pem"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/kiberdruzhinnik/go-exchange-api/constants"
	"github.com/kiberdruzhinnik/go-exchange-api/utils"
)

func TestHTTPClientIncludesRussianTrustedCertificates(t *testing.T) {
	client, err := utils.NewHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	wrapped, ok := client.Transport.(interface{ Unwrap() http.RoundTripper })
	if !ok {
		t.Fatalf("unexpected transport wrapper: %T", client.Transport)
	}
	transport, ok := wrapped.Unwrap().(*http.Transport)
	if !ok {
		t.Fatalf("unexpected transport: %T", client.Transport)
	}
	if transport.TLSClientConfig == nil || transport.TLSClientConfig.RootCAs == nil || transport.TLSClientConfig.InsecureSkipVerify {
		t.Fatal("expected certificate verification with an explicit trust store")
	}
	systemRoots, err := x509.SystemCertPool()
	if err != nil {
		t.Fatal(err)
	}
	subjects := transport.TLSClientConfig.RootCAs.Subjects()
	for _, subject := range systemRoots.Subjects() {
		if !containsSubject(subjects, subject) {
			t.Fatal("system certificate missing from trust store")
		}
	}
	rootPool := x509.NewCertPool()
	var sub *x509.Certificate
	for _, name := range []string{"russian_trusted_root_ca.crt", "russian_trusted_sub_ca.crt"} {
		data, err := os.ReadFile("../utils/certs/" + name)
		if err != nil {
			t.Fatal(err)
		}
		block, _ := pem.Decode(data)
		if block == nil {
			t.Fatalf("invalid PEM: %s", name)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			t.Fatal(err)
		}
		if !cert.IsCA || !containsSubject(subjects, cert.RawSubject) {
			t.Fatalf("CA missing from trust store: %s", name)
		}
		if name == "russian_trusted_root_ca.crt" {
			rootPool.AddCert(cert)
		} else {
			sub = cert
		}
	}
	if _, err := sub.Verify(x509.VerifyOptions{Roots: rootPool}); err != nil {
		t.Fatalf("invalid subordinate CA chain: %v", err)
	}
}

func containsSubject(subjects [][]byte, want []byte) bool {
	for _, subject := range subjects {
		if bytes.Equal(subject, want) {
			return true
		}
	}
	return false
}

func TestHTTPClientRejectsUntrustedServer(t *testing.T) {
	server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("unexpected success")) }))
	server.Config.ErrorLog = log.New(io.Discard, "", 0)
	server.StartTLS()
	defer server.Close()
	client, err := utils.NewHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := client.Get(server.URL)
	if resp != nil {
		resp.Body.Close()
	}
	if err == nil {
		t.Fatal("expected rejection of an untrusted server certificate")
	}
}

func TestHTTPClientSetsBrowserUserAgentAndPreservesRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.UserAgent(); got != constants.BrowserUserAgent {
			t.Errorf("User-Agent = %q, want %q", got, constants.BrowserUserAgent)
		}
		if r.URL.Path == "/redirect" {
			http.Redirect(w, r, "/final", http.StatusFound)
			return
		}
		w.Write([]byte("ok"))
	}))
	defer server.Close()
	client, err := utils.NewHTTPClient()
	if err != nil {
		t.Fatal(err)
	}
	for _, userAgent := range []string{"", "custom-agent"} {
		request, err := http.NewRequest(http.MethodGet, server.URL+"/redirect", nil)
		if err != nil {
			t.Fatal(err)
		}
		if userAgent != "" {
			request.Header.Set("User-Agent", userAgent)
		}
		response, err := client.Do(request)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if got := request.UserAgent(); got != userAgent {
			t.Errorf("caller request modified: User-Agent = %q, want %q", got, userAgent)
		}
	}
}

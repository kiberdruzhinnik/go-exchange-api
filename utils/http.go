package utils

import (
	"crypto/tls"
	"crypto/x509"
	"embed"
	"fmt"
	"net/http"
	"sync"
	"time"
)

//go:embed certs/*.crt
var trustedCertificates embed.FS

var trustedRoots = sync.OnceValues(func() (*x509.CertPool, error) {
	roots, err := x509.SystemCertPool()
	if err != nil {
		return nil, fmt.Errorf("load system certificates: %w", err)
	}
	for _, name := range []string{"russian_trusted_root_ca.crt", "russian_trusted_sub_ca.crt"} {
		pem, err := trustedCertificates.ReadFile("certs/" + name)
		if err != nil {
			return nil, fmt.Errorf("load bundled certificate: %w", err)
		}
		if !roots.AppendCertsFromPEM(pem) {
			return nil, fmt.Errorf("invalid bundled certificate: %s", name)
		}
	}
	return roots, nil
})

// Cache cloned transports to preserve connection pooling without modifying the
// process-wide transport or its TLS settings.
var trustedTransports sync.Map

// NewHTTPClient retains system CAs and adds the bundled Russian Trusted CAs.
// Custom default RoundTrippers are preserved; they control their own TLS setup.
func NewHTTPClient() (*http.Client, error) {
	transport := http.DefaultTransport
	if base, ok := transport.(*http.Transport); ok {
		roots, err := trustedRoots()
		if err != nil {
			return nil, err
		}
		if cached, ok := trustedTransports.Load(base); ok {
			transport = cached.(*http.Transport)
		} else {
			clone := base.Clone()
			if clone.TLSClientConfig == nil {
				clone.TLSClientConfig = &tls.Config{}
			}
			clone.TLSClientConfig.RootCAs = roots
			cached, _ := trustedTransports.LoadOrStore(base, clone)
			transport = cached.(*http.Transport)
		}
	}
	return &http.Client{Transport: transport, Timeout: 30 * time.Second}, nil
}

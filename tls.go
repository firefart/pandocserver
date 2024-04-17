package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

func (app *application) setupTLSConfig() (*tls.Config, error) {
	tlsConfig := &tls.Config{MinVersion: tls.VersionTLS13}

	if app.config.Server.RootCA != "" {
		caCertPEM, err := os.ReadFile(app.config.Server.RootCA)
		if err != nil {
			return nil, err
		}
		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(caCertPEM)
		if !ok {
			return nil, fmt.Errorf("failed to parse root certificate")
		}

		tlsConfig.ClientCAs = roots
		tlsConfig.ClientAuth = tls.RequireAndVerifyClientCert
	}

	if app.config.Server.CertSubject != "" {
		tlsConfig.VerifyPeerCertificate = func(_ [][]byte, verifiedChains [][]*x509.Certificate) error {
			certs := make(map[string]bool)
			// only loop over verified chains (matches the rootca)
			for _, x := range verifiedChains {
				// we need at least one certificate
				if len(x) == 0 {
					continue
				}
				// it seems like the first certificate is always the leaf
				// only add the leaf to the array as we want to check the leaf's subject only
				leafSubject := x[0].Subject.String()
				if _, ok := certs[leafSubject]; !ok {
					certs[leafSubject] = true
				}
				for _, y := range x {
					app.logger.Debug("Got certificate", slog.String("subject", y.Subject.String()), slog.Int64("serial", y.SerialNumber.Int64()))
				}
			}

			var subjects []string
			for subject := range certs {
				if subject == app.config.Server.CertSubject {
					app.logger.Debug("Allowing certificate", slog.String("subject", subject))
					// allow
					return nil
				}
				subjects = append(subjects, subject)
			}

			return fmt.Errorf("access denied, no valid certificate provided. Got the following subjects: %s", strings.Join(subjects, ", "))
		}
	}

	return tlsConfig, nil
}

package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"log"
)

var globCert *x509.Certificate

func makeTLSConfig(client bool) *tls.Config {
	certStore := x509.NewCertPool()
	block, _ := pem.Decode([]byte(serverCert))
	if block == nil {
		panic("Filed to parse pem server cert")
	}

	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		panic(err)
	}

	globCert = cert

	certStore.AddCert(cert)

	tlsCert, err := tls.X509KeyPair([]byte(serverCert), []byte(serverKey))
	if err != nil {
		panic(err)
	}

	var cfg *tls.Config
	if client {
		cfg = &tls.Config{
			RootCAs:            certStore,
			Certificates:       []tls.Certificate{tlsCert},
			ServerName:         cert.Subject.CommonName,
			InsecureSkipVerify: true,
			VerifyPeerCertificate: func(rawCerts [][]byte, verifiedChains [][]*x509.Certificate) error {
				sentCerts := x509.NewCertPool()
				for _, rawCert := range rawCerts {
					cert, _ := x509.ParseCertificate(rawCert)
					sentCerts.AddCert(cert)
				}
				opts := x509.VerifyOptions{
					Roots: sentCerts,
				}
				_, err := globCert.Verify(opts)
				return err
			},
		}
	} else {
		cfg = &tls.Config{
			ClientCAs:    certStore,
			RootCAs:      certStore,
			Certificates: []tls.Certificate{tlsCert},
			ClientAuth:   tls.RequireAndVerifyClientCert,
			ServerName:   cert.Subject.CommonName,
		}
	}
	cfg.MinVersion = tls.VersionTLS12

	log.Println(cfg.ServerName, len(cfg.ServerName))

	return cfg
}

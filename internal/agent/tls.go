package agent

import (
	"crypto/tls"
	"log"
	"net/url"
	"strings"
)

func tlsConfigForControlPlane(baseURL, dataDir, nodeID string, logger *log.Logger) *tls.Config {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil
	}
	switch strings.ToLower(u.Scheme) {
	case "http", "":
		return nil
	case "https":
	default:
		return nil
	}
	cert, err := ensureIdentity(dataDir, nodeID)
	if err != nil {
		if logger != nil {
			logger.Printf("mTLS identity unavailable: %v", err)
		}
		return nil
	}
	return &tls.Config{
		Certificates: []tls.Certificate{*cert},
		MinVersion:   tls.VersionTLS12,
	}
}

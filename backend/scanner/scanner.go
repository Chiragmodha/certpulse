package scanner

import (
	"context"
	"crypto/ecdsa"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"time"

	"certpulse/backend/db"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ScanResult struct {
	CommonName             string
	SubjectAlternativeNames []string
	IssuerOrganization     string
	SerialNumber           string
	ValidFrom              time.Time
	ValidTo                time.Time
	SignatureAlgorithm     string
	KeyAlgorithm           string
	KeySize                int
	ChainValid             bool
	RawPEM                 string
}

// ScanDomain dials the domain on the given port and inspects the TLS certificate.
func ScanDomain(domain string, port int) (*ScanResult, error) {
	address := fmt.Sprintf("%s:%d", domain, port)
	dialer := &net.Dialer{
		Timeout: 5 * time.Second,
	}

	// Dial connection with TLS config
	conn, err := tls.DialWithDialer(dialer, "tcp", address, &tls.Config{
		ServerName:         domain,
		InsecureSkipVerify: true, // skip verify so we can read invalid/expired cert details
	})
	if err != nil {
		return nil, fmt.Errorf("tls connection failed: %w", err)
	}
	defer conn.Close()

	state := conn.ConnectionState()
	if len(state.PeerCertificates) == 0 {
		return nil, fmt.Errorf("no peer certificates returned")
	}

	// Leaf cert is always the first one in the list
	leaf := state.PeerCertificates[0]

	// Check if the chain validates against root system CAs
	chainValid := true
	roots, err := x509.SystemCertPool()
	if err == nil {
		opts := x509.VerifyOptions{
			DNSName:       domain,
			Intermediates: x509.NewCertPool(),
			Roots:         roots,
		}
		for i := 1; i < len(state.PeerCertificates); i++ {
			opts.Intermediates.AddCert(state.PeerCertificates[i])
		}
		if _, err := leaf.Verify(opts); err != nil {
			chainValid = false
		}
	} else {
		chainValid = false
	}

	// Detect key specs
	keySize := 0
	switch pub := leaf.PublicKey.(type) {
	case *rsa.PublicKey:
		keySize = pub.N.BitLen()
	case *ecdsa.PublicKey:
		keySize = pub.Params().BitSize
	}

	result := &ScanResult{
		CommonName:         leaf.Subject.CommonName,
		IssuerOrganization: fmt.Sprintf("%v", leaf.Issuer.Organization),
		SerialNumber:       leaf.SerialNumber.String(),
		ValidFrom:          leaf.NotBefore,
		ValidTo:            leaf.NotAfter,
		SignatureAlgorithm: leaf.SignatureAlgorithm.String(),
		KeyAlgorithm:       leaf.PublicKeyAlgorithm.String(),
		ChainValid:         chainValid,
		RawPEM:             "", // can construct PEM if needed, or leave blank/placeholder
	}

	result.SubjectAlternativeNames = leaf.DNSNames

	return result, nil
}

// TriggerScan runs a scan on a specific monitored endpoint and updates the database.
func TriggerScan(ctx context.Context, endpointID string) error {
	var domain string
	var port int
	var workspaceID string

	// Query endpoint info
	err := db.Pool.QueryRow(ctx, 
		"SELECT domain_name, port, workspace_id FROM monitored_endpoints WHERE id = $1", 
		endpointID,
	).Scan(&domain, &port, &workspaceID)
	if err != nil {
		return fmt.Errorf("failed to fetch endpoint: %w", err)
	}

	scanResult, scanErr := ScanDomain(domain, port)
	if scanErr != nil {
		// Log error and update status to error/unreachable
		_, dbErr := db.Pool.Exec(ctx, 
			"UPDATE monitored_endpoints SET last_scan_status = 'unreachable', last_scan_at = $1 WHERE id = $2",
			time.Now(), endpointID,
		)
		if dbErr != nil {
			return fmt.Errorf("scan failed (%v), db update failed: %w", scanErr, dbErr)
		}
		return fmt.Errorf("domain scanning failed: %w", scanErr)
	}

	// Save or update certificate details in certificates table
	var certID string
	err = db.Pool.QueryRow(ctx, `
		INSERT INTO certificates (
			workspace_id, common_name, subject_alternative_names, issuer_organization, 
			serial_number, valid_from, valid_to, signature_algorithm, key_algorithm, 
			key_size, chain_valid
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id
	`, 
		workspaceID, scanResult.CommonName, scanResult.SubjectAlternativeNames, 
		scanResult.IssuerOrganization, scanResult.SerialNumber, scanResult.ValidFrom, 
		scanResult.ValidTo, scanResult.SignatureAlgorithm, scanResult.KeyAlgorithm, 
		scanResult.KeySize, scanResult.ChainValid,
	).Scan(&certID)
	if err != nil {
		return fmt.Errorf("failed to save certificate: %w", err)
	}

	// Determine overall status
	status := "healthy"
	daysLeft := time.Until(scanResult.ValidTo).Hours() / 24
	if daysLeft <= 0 {
		status = "expired"
	} else if daysLeft <= 30 {
		status = "expiring"
	}

	if !scanResult.ChainValid {
		status = "error" // chain validation error
	}

	// Update monitored endpoint with cert pointer and status
	_, err = db.Pool.Exec(ctx, `
		UPDATE monitored_endpoints 
		SET active_certificate_id = $1, last_scan_status = $2, last_scan_at = $3 
		WHERE id = $4
	`, certID, status, time.Now(), endpointID)
	if err != nil {
		return fmt.Errorf("failed to update monitored endpoint status: %w", err)
	}

	return nil
}

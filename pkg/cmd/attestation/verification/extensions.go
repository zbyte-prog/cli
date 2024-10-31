package verification

import (
	"errors"
	"fmt"
	"strings"
)

var (
	GitHubOIDCIssuer       = "https://token.actions.githubusercontent.com"
	GitHubTenantOIDCIssuer = "https://token.actions.%s.ghe.com"
)

func VerifyCertExtensions(results []*AttestationProcessingResult, ec EnforcementCriteria) error {
	if len(results) == 0 {
		return errors.New("no attestations proccessing results")
	}

	var atLeastOneVerified bool
	for _, attestation := range results {
		if err := verifyCertExtensions(attestation, ec); err != nil {
			return err
		}
		atLeastOneVerified = true
	}

	if atLeastOneVerified {
		return nil
	} else {
		return ErrNoAttestationsVerified
	}
}

func verifyCertExtensions(verifiedCert certificate.Summary, criteria EnforcementCriteria) error {
	if c.Extensions.SourceRepositoryOwnerURI != "" {
		sourceRepositoryOwnerURI := attestation.VerificationResult.Signature.Certificate.Extensions.SourceRepositoryOwnerURI
		if !strings.EqualFold(c.Extensions.SourceRepositoryOwnerURI, sourceRepositoryOwnerURI) {
			return fmt.Errorf("expected SourceRepositoryOwnerURI to be %s, got %s", c.Extensions.SourceRepositoryOwnerURI, sourceRepositoryOwnerURI)
		}
	}

	// if repo is set, check the SourceRepositoryURI field
	if c.Extensions.SourceRepositoryURI != "" {
		sourceRepositoryURI := attestation.VerificationResult.Signature.Certificate.Extensions.SourceRepositoryURI
		if !strings.EqualFold(c.Extensions.SourceRepositoryURI, sourceRepositoryURI) {
			return fmt.Errorf("expected SourceRepositoryURI to be %s, got %s", c.Extensions.SourceRepositoryURI, sourceRepositoryURI)
		}
	}

	// if issuer is anything other than the default, use the user-provided value;
	// otherwise, select the appropriate default based on the tenant
	if c.OIDCIssuer != "" {
		certIssuer := attestation.VerificationResult.Signature.Certificate.Extensions.Issuer
		if !strings.EqualFold(c.OIDCIssuer, certIssuer) {
			if strings.Index(certIssuer, c.OIDCIssuer+"/") == 0 {
				return fmt.Errorf("expected Issuer to be %s, got %s -- if you have a custom OIDC issuer policy for your enterprise, use the --cert-oidc-issuer flag with your expected issuer", c.OIDCIssuer, certIssuer)
			}
			return fmt.Errorf("expected Issuer to be %s, got %s", c.OIDCIssuer, certIssuer)
		}
	}

	return nil
}

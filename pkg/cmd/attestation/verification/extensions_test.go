package verification

import (
	"testing"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/stretchr/testify/require"
)

func TestVerifyCertExtensions(t *testing.T) {
	results := []*AttestationProcessingResult{
		{
			VerificationResult: &verify.VerificationResult{
				Signature: &verify.SignatureVerificationResult{
					Certificate: &certificate.Summary{
						Extensions: certificate.Extensions{
							SourceRepositoryOwnerURI: "https://github.com/owner",
							SourceRepositoryURI:      "https://github.com/owner/repo",
							Issuer:                   "https://token.actions.githubusercontent.com",
						},
					},
				},
			},
		},
	}

	c := EnforcementCriteria{
		Extensions: Extensions{
			SourceRepositoryOwnerURI: "https://github.com/owner",
			SourceRepositoryURI:      "https://github.com/owner/repo",
		},
		OIDCIssuer: GitHubOIDCIssuer,
	}

	t.Run("success", func(t *testing.T) {
		err := VerifyCertExtensions(results, c)
		require.NoError(t, err)
	})

	t.Run("with wrong SourceRepositoryOwnerURI", func(t *testing.T) {
		expectedCriteria := c
		expectedCriteria.Extensions.SourceRepositoryOwnerURI = "https://github.com/wrong"
		err := VerifyCertExtensions(results, expectedCriteria)
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://github.com/owner, got https://github.com/wrong")
	})

	t.Run("with wrong SourceRepositoryURI", func(t *testing.T) {
		expectedCriteria := c
		expectedCriteria.Extensions.SourceRepositoryURI = "https://github.com/foo/wrong"
		err := VerifyCertExtensions(results, expectedCriteria)
		require.ErrorContains(t, err, "expected SourceRepositoryURI to be https://github.com/owner/wrong, got https://github.com/wrong/bar")
	})

	t.Run("with wrong OIDCIssuer", func(t *testing.T) {
		expectedCriteria := c
		expectedCriteria.OIDCIssuer = "wrong"
		err := VerifyCertExtensions(results, expectedCriteria)
		require.ErrorContains(t, err, "expected Issuer to be wrong, got https://token.actions.githubusercontent.com")
	})

	t.Run("with partial OIDCIssuer match", func(t *testing.T) {
		expectedCriteria := c
		expectedCriteria.OIDCIssuer = "https://token.actions.githubusercontent.com"
		err := VerifyCertExtensions(results, expectedCriteria)
		require.ErrorContains(t, err, "expected Issuer to be https://token.actions.githubusercontent.com, got https://token.actions.githubusercontent.com/foo-bar -- if you have a custom OIDC issuer")
	})
}

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
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://github.com/wrong, got https://github.com/owner")
	})

	t.Run("with wrong SourceRepositoryURI", func(t *testing.T) {
		expectedCriteria := c
		expectedCriteria.Extensions.SourceRepositoryURI = "https://github.com/foo/wrong"
		err := VerifyCertExtensions(results, expectedCriteria)
		require.ErrorContains(t, err, "expected SourceRepositoryURI to be https://github.com/foo/wrong, got https://github.com/owner/repo")
	})

	t.Run("with wrong OIDCIssuer", func(t *testing.T) {
		expectedCriteria := c
		expectedCriteria.OIDCIssuer = "wrong"
		err := VerifyCertExtensions(results, expectedCriteria)
		require.ErrorContains(t, err, "expected Issuer to be wrong, got https://token.actions.githubusercontent.com")
	})

	t.Run("with partial OIDCIssuer match", func(t *testing.T) {
		expectedResults := results
		expectedResults[0].VerificationResult.Signature.Certificate.Extensions.Issuer = "https://token.actions.githubusercontent.com/foo-bar"
		err := VerifyCertExtensions(expectedResults, c)
		require.ErrorContains(t, err, "expected Issuer to be https://token.actions.githubusercontent.com, got https://token.actions.githubusercontent.com/foo-bar -- if you have a custom OIDC issuer")
	})
}

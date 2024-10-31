package verification

import (
	"testing"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/stretchr/testify/require"
)

func createSampleResult() *AttestationProcessingResult {
	return &AttestationProcessingResult{
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
	}
}

func TestVerifyCertExtensions(t *testing.T) {
	results := []*AttestationProcessingResult{createSampleResult()}

	t.Run("passes with one result", func(t *testing.T) {
		err := VerifyCertExtensions(results, "", "owner", "owner/repo", GitHubOIDCIssuer)
		require.NoError(t, err)
	})

	t.Run("passes with 1/2 valid results", func(t *testing.T) {
		twoResults := []*AttestationProcessingResult{createSampleResult(), createSampleResult()}
		require.Len(t, twoResults, 2)
		twoResults[1].VerificationResult.Signature.Certificate.Extensions.SourceRepositoryOwnerURI = "https://github.com/wrong"

		err := VerifyCertExtensions(twoResults, "", "owner", "owner/repo", GitHubOIDCIssuer)
		require.NoError(t, err)
	})

	t.Run("fails when all results fail verification", func(t *testing.T) {
		twoResults := []*AttestationProcessingResult{createSampleResult(), createSampleResult()}
		require.Len(t, twoResults, 2)
		twoResults[0].VerificationResult.Signature.Certificate.Extensions.SourceRepositoryOwnerURI = "https://github.com/wrong"
		twoResults[1].VerificationResult.Signature.Certificate.Extensions.SourceRepositoryOwnerURI = "https://github.com/wrong"

		err := VerifyCertExtensions(twoResults, "", "owner", "owner/repo", GitHubOIDCIssuer)
		require.Error(t, err)
	})
}

func TestVerifyCertExtension(t *testing.T) {
	t.Run("with owner and repo, but wrong tenant", func(t *testing.T) {
		err := verifyCertExtension(createSampleResult(), "foo", "owner", "owner/repo", GitHubOIDCIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://foo.ghe.com/owner, got https://github.com/owner")
	})

	t.Run("with owner", func(t *testing.T) {
		err := verifyCertExtension(createSampleResult(), "", "owner", "", GitHubOIDCIssuer)
		require.NoError(t, err)
	})

	t.Run("with wrong owner", func(t *testing.T) {
		err := verifyCertExtension(createSampleResult(), "", "wrong", "", GitHubOIDCIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://github.com/wrong, got https://github.com/owner")
	})

	t.Run("with wrong repo", func(t *testing.T) {
		err := verifyCertExtension(createSampleResult(), "", "owner", "wrong", GitHubOIDCIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryURI to be https://github.com/wrong, got https://github.com/owner/repo")
	})

	t.Run("with wrong issuer", func(t *testing.T) {
		err := verifyCertExtension(createSampleResult(), "", "owner", "", "wrong")
		require.ErrorContains(t, err, "expected Issuer to be wrong, got https://token.actions.githubusercontent.com")
	})
}

func TestVerifyCertExtensionCustomizedIssuer(t *testing.T) {
	result := &AttestationProcessingResult{
		VerificationResult: &verify.VerificationResult{
			Signature: &verify.SignatureVerificationResult{
				Certificate: &certificate.Summary{
					Extensions: certificate.Extensions{
						SourceRepositoryOwnerURI: "https://github.com/owner",
						SourceRepositoryURI:      "https://github.com/owner/repo",
						Issuer:                   "https://token.actions.githubusercontent.com/foo-bar",
					},
				},
			},
		},
	}

	t.Run("with exact issuer match", func(t *testing.T) {
		err := verifyCertExtension(result, "", "owner", "owner/repo", "https://token.actions.githubusercontent.com/foo-bar")
		require.NoError(t, err)
	})

	t.Run("with partial issuer match", func(t *testing.T) {
		err := verifyCertExtension(result, "", "owner", "owner/repo", "https://token.actions.githubusercontent.com")
		require.ErrorContains(t, err, "expected Issuer to be https://token.actions.githubusercontent.com, got https://token.actions.githubusercontent.com/foo-bar -- if you have a custom OIDC issuer")
	})

	t.Run("with wrong issuer", func(t *testing.T) {
		err := verifyCertExtension(result, "", "owner", "", "wrong")
		require.ErrorContains(t, err, "expected Issuer to be wrong, got https://token.actions.githubusercontent.com/foo-bar")
	})
}

func TestVerifyTenancyCertExtensions(t *testing.T) {
	defaultIssuer := GitHubOIDCIssuer

	result := &AttestationProcessingResult{
		VerificationResult: &verify.VerificationResult{
			Signature: &verify.SignatureVerificationResult{
				Certificate: &certificate.Summary{
					Extensions: certificate.Extensions{
						SourceRepositoryOwnerURI: "https://foo.ghe.com/owner",
						SourceRepositoryURI:      "https://foo.ghe.com/owner/repo",
						Issuer:                   "https://token.actions.foo.ghe.com",
					},
				},
			},
		},
	}

	t.Run("with owner and repo", func(t *testing.T) {
		err := verifyCertExtension(result, "foo", "owner", "owner/repo", defaultIssuer)
		require.NoError(t, err)
	})

	t.Run("with owner and repo, no tenant", func(t *testing.T) {
		err := verifyCertExtension(result, "", "owner", "owner/repo", defaultIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://github.com/owner, got https://foo.ghe.com/owner")
	})

	t.Run("with owner and repo, wrong tenant", func(t *testing.T) {
		err := verifyCertExtension(result, "bar", "owner", "owner/repo", defaultIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://bar.ghe.com/owner, got https://foo.ghe.com/owner")
	})

	t.Run("with owner", func(t *testing.T) {
		err := verifyCertExtension(result, "foo", "owner", "", defaultIssuer)
		require.NoError(t, err)
	})

	t.Run("with wrong owner", func(t *testing.T) {
		err := verifyCertExtension(result, "foo", "wrong", "", defaultIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryOwnerURI to be https://foo.ghe.com/wrong, got https://foo.ghe.com/owner")
	})

	t.Run("with wrong repo", func(t *testing.T) {
		err := verifyCertExtension(result, "foo", "owner", "wrong", defaultIssuer)
		require.ErrorContains(t, err, "expected SourceRepositoryURI to be https://foo.ghe.com/wrong, got https://foo.ghe.com/owner/repo")
	})

	t.Run("with correct, non-default issuer", func(t *testing.T) {
		err := verifyCertExtension(result, "foo", "owner", "owner/repo", "https://token.actions.foo.ghe.com")
		require.NoError(t, err)
	})

	t.Run("with wrong issuer", func(t *testing.T) {
		err := verifyCertExtension(result, "foo", "owner", "owner/repo", "wrong")
		require.ErrorContains(t, err, "expected Issuer to be wrong, got https://token.actions.foo.ghe.com")
	})
}

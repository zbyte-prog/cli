package verify

import (
	"fmt"

	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact"
	"github.com/cli/cli/v2/pkg/cmd/attestation/verification"
	"github.com/sigstore/sigstore-go/pkg/verify"
)

func getAttestations(o *Options, a artifact.DigestedArtifact) ([]*api.Attestation, string, error) {
	if o.BundlePath != "" {
		attestations, err := verification.GetLocalAttestations(o.BundlePath)
		if err != nil {
			msg := fmt.Sprintf("✗ Loading attestations from %s failed", a.URL)
			return nil, msg, err
		}
		pluralAttestation := text.Pluralize(len(attestations), "attestation")
		msg := fmt.Sprintf("Loaded %s from %s", pluralAttestation, o.BundlePath)
		return attestations, msg, nil
	}

	if o.UseBundleFromRegistry {
		attestations, err := verification.GetOCIAttestations(o.OCIClient, a)
		if err != nil {
			msg := "✗ Loading attestations from OCI registry failed"
			return nil, msg, err
		}
		pluralAttestation := text.Pluralize(len(attestations), "attestation")
		msg := fmt.Sprintf("Loaded %s from %s", pluralAttestation, o.ArtifactPath)
		return attestations, msg, nil
	}

	params := verification.FetchRemoteAttestationsParams{
		Digest: a.DigestWithAlg(),
		Limit:  o.Limit,
		Owner:  o.Owner,
		Repo:   o.Repo,
	}

	attestations, err := verification.GetRemoteAttestations(o.APIClient, params)
	if err != nil {
		msg := "✗ Loading attestations from GitHub API failed"
		return nil, msg, err
	}
	pluralAttestation := text.Pluralize(len(attestations), "attestation")
	msg := fmt.Sprintf("Loaded %s from GitHub API", pluralAttestation)
	return attestations, msg, nil
}

func verifyAttestations(attestations []*api.Attestation, sgVerifier verification.SigstoreVerifier, sgPolicy verify.PolicyBuilder, ec verification.EnforcementCriteria) ([]*verification.AttestationProcessingResult, string, error) {
	sigstoreVerified, err := sgVerifier.Verify(attestations, sgPolicy)
	if err != nil {
		errMsg := "✗ Sigstore verification failed"
		return nil, errMsg, err
	}

	// Verify extensions
	certExtVerified, err := verification.VerifyCertExtensions(sigstoreVerified, ec)
	if err != nil {
		errMsg := "✗ Policy verification failed"
		return nil, errMsg, err
	}

	return certExtVerified, "", nil
}

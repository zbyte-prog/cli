package verify

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact"
	"github.com/cli/cli/v2/pkg/cmd/attestation/verification"
)

const (
	// represents the GitHub hosted runner in the certificate RunnerEnvironment extension
	GitHubRunner = "github-hosted"
	hostRegex    = `^[a-zA-Z0-9-]+\.[a-zA-Z0-9-]+.*$`
)

type ExpectedExtensions struct {
	RunnerEnvironment string
	SANRegex          string
	SAN               string
	BuildSourceRepo   string
	SignerWorkflow    string
}

type SigstoreInstance string

const (
	PublicGood SigstoreInstance = "public-good"
	GitHub     SigstoreInstance = "github"
	Custom     SigstoreInstance = "custom"
)

type Policy struct {
	ExpectedExtensions       ExpectedExtensions
	ExpectedPredicateType    string
	ExpectedSigstoreInstance string
	Artifact                 artifact.DigestedArtifact
}

func newPolicy(opts *Options, a artifact.DigestedArtifact) (Policy, error) {
	p := Policy{}

	if opts.SignerRepo != "" {
		signedRepoRegex := expandToGitHubURL(opts.Tenant, opts.SignerRepo)
		p.ExpectedExtensions.SANRegex = signedRepoRegex
	} else if opts.SignerWorkflow != "" {
		validatedWorkflowRegex, err := validateSignerWorkflow(opts)
		if err != nil {
			return Policy{}, err
		}

		p.ExpectedExtensions.SANRegex = validatedWorkflowRegex
	} else {
		p.ExpectedExtensions.SANRegex = opts.SANRegex
		p.ExpectedExtensions.SAN = opts.SAN
	}

	if opts.DenySelfHostedRunner {
		p.ExpectedExtensions.RunnerEnvironment = GitHubRunner
	} else {
		p.ExpectedExtensions.RunnerEnvironment = "*"
	}

	if opts.Repo != "" {
		if opts.Tenant != "" {
			p.ExpectedExtensions.BuildSourceRepo = fmt.Sprintf("https://%s.ghe.com/%s", opts.Tenant, opts.Repo)
		}
		p.ExpectedExtensions.BuildSourceRepo = fmt.Sprintf("https://github.com/%s", opts.Repo)
	}
	return p, nil
}

func (p *Policy) Verify(a []*api.Attestation) (bool, string) {
	filtered := verification.FilterAttestations(p.ExpectedPredicateType, a)
	if len(filtered) == 0 {
		return false, fmt.Sprintf("✗ No attestations found with predicate type: %s\n", p.ExpectedPredicateType)
	}

	return true, ""
}

func expandToGitHubURL(tenant, ownerOrRepo string) string {
	if tenant == "" {
		return fmt.Sprintf("(?i)^https://github.com/%s/", ownerOrRepo)
	}
	return fmt.Sprintf("(?i)^https://%s.ghe.com/%s/", tenant, ownerOrRepo)
}

func (p *Policy) buildCertificateIdentityOption() (verify.PolicyOption, error) {
	sanMatcher, err := verify.NewSANMatcher(p.ExpectedExtensions.SAN, p.ExpectedExtensions.SANRegex)
	if err != nil {
		return nil, err
	}

	// Accept any issuer, we will verify the issuer as part of the extension verification
	issuerMatcher, err := verify.NewIssuerMatcher("", ".*")
	if err != nil {
		return nil, err
	}

	extensions := certificate.Extensions{
		RunnerEnvironment: p.ExpectedExtensions.RunnerEnvironment,
	}

	certId, err := verify.NewCertificateIdentity(sanMatcher, issuerMatcher, extensions)
	if err != nil {
		return nil, err
	}

	return verify.WithCertificateIdentity(certId), nil
}

func (p *Policy) SigstorePolicy() (verify.PolicyBuilder, error) {
	artifactDigestPolicyOption, err := verification.BuildDigestPolicyOption(p.Artifact)
	if err != nil {
		return verify.PolicyBuilder{}, err
	}

	certIdOption, err := p.buildCertificateIdentityOption()
	if err != nil {
		return verify.PolicyBuilder{}, err
	}

	policy := verify.NewPolicy(artifactDigestPolicyOption, certIdOption)
	return policy, nil
}

func addSchemeToRegex(s string) string {
	return fmt.Sprintf("^https://%s", s)
}

func validateSignerWorkflow(opts *Options) (string, error) {
	// we expect a provided workflow argument be in the format [HOST/]/<OWNER>/<REPO>/path/to/workflow.yml
	// if the provided workflow does not contain a host, set the host
	match, err := regexp.MatchString(hostRegex, opts.SignerWorkflow)
	if err != nil {
		return "", err
	}

	if match {
		return fmt.Sprintf("^https://%s", opts.SignerWorkflow), nil
	}

	if opts.Hostname == "" {
		return "", errors.New("unknown host")
	}

	return fmt.Sprintf("^https://%s/%s/%s", opts.Hostname, opts.SignerWorkflow), nil
}

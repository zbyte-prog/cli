package verify

import (
	"errors"
	"fmt"
	"regexp"

	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/cli/cli/v2/pkg/cmd/attestation/artifact"
	"github.com/cli/cli/v2/pkg/cmd/attestation/verification"
)

const (
	// represents the GitHub hosted runner in the certificate RunnerEnvironment extension
	GitHubRunner = "github-hosted"
	hostRegex    = `^[a-zA-Z0-9-]+\.[a-zA-Z0-9-]+.*$`
)

func expandToGitHubURL(tenant, ownerOrRepo string) string {
	if tenant == "" {
		return fmt.Sprintf("(?i)^https://github.com/%s/", ownerOrRepo)
	}
	return fmt.Sprintf("(?i)^https://%s.ghe.com/%s/", tenant, ownerOrRepo)
}

func newEnforcementCriteria(opts *Options) (verification.EnforcementCriteria, error) {
	c := verification.EnforcementCriteria{}

	if opts.SignerRepo != "" {
		signedRepoRegex := expandToGitHubURL(opts.Tenant, opts.SignerRepo)
		c.Extensions.SANRegex = signedRepoRegex
	} else if opts.SignerWorkflow != "" {
		validatedWorkflowRegex, err := validateSignerWorkflow(opts)
		if err != nil {
			return verification.EnforcementCriteria{}, err
		}

		c.Extensions.SANRegex = validatedWorkflowRegex
	} else {
		c.Extensions.SANRegex = opts.SANRegex
		c.Extensions.SAN = opts.SAN
	}

	if opts.DenySelfHostedRunner {
		c.Extensions.RunnerEnvironment = GitHubRunner
	}

	if opts.Repo != "" {
		if opts.Tenant != "" {
			c.Extensions.SourceRepositoryURI = fmt.Sprintf("https://%s.ghe.com/%s", opts.Tenant, opts.Repo)
		} else {
			c.Extensions.SourceRepositoryURI = fmt.Sprintf("https://github.com/%s", opts.Repo)
		}
	}

	if opts.Tenant != "" {
		c.Extensions.SourceRepositoryOwnerURI = fmt.Sprintf("https://%s.ghe.com/%s", opts.Tenant, opts.Owner)
	} else {
		c.Extensions.SourceRepositoryOwnerURI = fmt.Sprintf("https://github.com/%s", opts.Owner)
	}

	// if tenant is provided, select the appropriate default based on the tenant
	// otherwise, use the provided OIDCIssuer
	if opts.Tenant != "" {
		c.OIDCIssuer = fmt.Sprintf(verification.GitHubTenantOIDCIssuer, opts.Tenant)
	} else {
		c.OIDCIssuer = opts.OIDCIssuer
	}

	return c, nil
}

func buildCertificateIdentityOption(c verification.EnforcementCriteria) (verify.PolicyOption, error) {
	sanMatcher, err := verify.NewSANMatcher(c.Extensions.SAN, c.Extensions.SANRegex)
	if err != nil {
		return nil, err
	}

	// Accept any issuer, we will verify the issuer as part of the extension verification
	issuerMatcher, err := verify.NewIssuerMatcher("", ".*")
	if err != nil {
		return nil, err
	}

	extensions := certificate.Extensions{
		RunnerEnvironment: c.Extensions.RunnerEnvironment,
	}

	certId, err := verify.NewCertificateIdentity(sanMatcher, issuerMatcher, extensions)
	if err != nil {
		return nil, err
	}

	return verify.WithCertificateIdentity(certId), nil
}

func buildSigstoreVerifyPolicy(c verification.EnforcementCriteria, a artifact.DigestedArtifact) (verify.PolicyBuilder, error) {
	artifactDigestPolicyOption, err := verification.BuildDigestPolicyOption(a)
	if err != nil {
		return verify.PolicyBuilder{}, err
	}

	certIdOption, err := buildCertificateIdentityOption(c)
	if err != nil {
		return verify.PolicyBuilder{}, err
	}

	policy := verify.NewPolicy(artifactDigestPolicyOption, certIdOption)
	return policy, nil
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

	return fmt.Sprintf("^https://%s/%s", opts.Hostname, opts.SignerWorkflow), nil
}

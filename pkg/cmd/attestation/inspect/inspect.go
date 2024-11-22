package inspect

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/cli/cli/v2/internal/ghinstance"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/cmd/attestation/api"
	"github.com/cli/cli/v2/pkg/cmd/attestation/io"
	"github.com/cli/cli/v2/pkg/cmd/attestation/verification"
	"github.com/cli/cli/v2/pkg/cmdutil"
	ghauth "github.com/cli/go-gh/v2/pkg/auth"
	"github.com/digitorus/timestamp"
	in_toto "github.com/in-toto/attestation/go/v1"
	"github.com/sigstore/sigstore-go/pkg/bundle"
	"github.com/sigstore/sigstore-go/pkg/fulcio/certificate"
	"github.com/sigstore/sigstore-go/pkg/verify"

	"github.com/MakeNowJust/heredoc"
	"github.com/spf13/cobra"
)

func NewInspectCmd(f *cmdutil.Factory, runF func(*Options) error) *cobra.Command {
	opts := &Options{}
	inspectCmd := &cobra.Command{
		Use:    "inspect [<file path> | oci://<OCI image URI>] --bundle <path-to-bundle>",
		Args:   cmdutil.ExactArgs(1, "must specify file path or container image URI, as well --bundle"),
		Hidden: true,
		Short:  "Inspect a sigstore bundle",
		Long: heredoc.Docf(`
			### NOTE: This feature is currently in public preview, and subject to change.

			Inspect a Sigstore bundle that has been downloaded to disk. See the %[1]sdownload%[1]s
			command.

			// The command requires either:
			// * a relative path to a local artifact, or
			// * a container image URI (e.g. %[1]soci://<my-OCI-image-URI>%[1]s)

			// Note that if you provide an OCI URI for the artifact you must already
			// be authenticated with a container registry.

			// The command also requires the %[1]s--bundle%[1]s flag, which provides a file
			// path to a previously downloaded Sigstore bundle. (See also the %[1]sdownload%[1]s
			// command).

			By default, the command will print information about the bundle in a table format.
			If the %[1]s--json-result%[1]s flag is provided, the command will print the
			information in JSON format.
		`, "`"),
		Example: heredoc.Doc(`
			# Inspect a Sigstore bundle and print the results in table format
			$ gh attestation inspect <path-to-bundle>

			# Inspect a Sigstore bundle and print the results in JSON format
			$ gh attestation inspect <path-to-bundle> --json-result

			// # Inspect a Sigsore bundle for an OCI artifact, and print the results in table format
			// $ gh attestation inspect oci://<my-OCI-image> --bundle <path-to-bundle>
		`),
		PreRunE: func(cmd *cobra.Command, args []string) error {
			// Create a logger for use throughout the inspect command
			opts.Logger = io.NewHandler(f.IOStreams)

			// set the artifact path
			opts.BundlePath = args[0]

			// Clean file path options
			opts.Clean()

			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// opts.OCIClient = oci.NewLiveClient()
			// if opts.Hostname == "" {
			// 	opts.Hostname, _ = ghauth.DefaultHost()
			// }
			//
			// if err := auth.IsHostSupported(opts.Hostname); err != nil {
			// 	return err
			// }
			//
			if runF != nil {
				return runF(opts)
			}

			config := verification.SigstoreConfig{
				Logger: opts.Logger,
			}

			// fetch the correct trust domain so we can verify the bundle
			if ghauth.IsTenancy(opts.Hostname) {
				hc, err := f.HttpClient()
				if err != nil {
					return err
				}
				apiClient := api.NewLiveClient(hc, opts.Hostname, opts.Logger)
				td, err := apiClient.GetTrustDomain()
				if err != nil {
					return err
				}
				tenant, found := ghinstance.TenantName(opts.Hostname)
				if !found {
					return fmt.Errorf("Invalid hostname provided: '%s'",
						opts.Hostname)
				}

				config.TrustDomain = td
				opts.Tenant = tenant
			}

			opts.SigstoreVerifier = verification.NewLiveSigstoreVerifier(config)

			if err := runInspect(opts); err != nil {
				return fmt.Errorf("Failed to inspect the artifact and bundle: %w", err)
			}
			return nil
		},
	}

	inspectCmd.Flags().StringVarP(&opts.BundlePath, "bundle", "b", "", "Path to bundle on disk, either a single bundle in a JSON file or a JSON lines file with multiple bundles")
	// inspectCmd.MarkFlagRequired("bundle") //nolint:errcheck
	inspectCmd.Flags().StringVarP(&opts.Hostname, "hostname", "", "", "Configure host to use")
	cmdutil.StringEnumFlag(inspectCmd, &opts.DigestAlgorithm, "digest-alg", "d", "sha256", []string{"sha256", "sha512"}, "The algorithm used to compute a digest of the artifact")
	cmdutil.AddFormatFlags(inspectCmd, &opts.exporter)

	return inspectCmd
}

type BundleInspectResult struct {
	InspectedBundles []BundleInspection `json:"inspectedBundles"`
}

type BundleInspection struct {
	Authentic              bool                  `json:"authentic"`
	Certificate            CertificateInspection `json:"certificate"`
	TransparencyLogEntries []TlogEntryInspection `json:"transparencyLogEntries"`
	SignedTimestamps       []time.Time           `json:"signedTimestamps"`
	Statement              in_toto.Statement     `json:"statement"`
}

type CertificateInspection struct {
	certificate.Summary
	NotBefore time.Time `json:"notBefore"`
	NotAfter  time.Time `json:"notAfter"`
}

type TlogEntryInspection struct {
	IntegratedTime time.Time
	LogID          string
}

func runInspect(opts *Options) error {
	attestations, err := verification.GetLocalAttestations(opts.BundlePath)
	if err != nil {
		return fmt.Errorf("failed to read attestations")
	}

	inspectedBundles := []BundleInspection{}
	sigstorePolicy := verify.NewPolicy(verify.WithoutArtifactUnsafe(), verify.WithoutIdentitiesUnsafe())

	for _, a := range attestations {
		inspectedBundle := BundleInspection{}

		_, err := opts.SigstoreVerifier.Verify([]*api.Attestation{a}, sigstorePolicy)
		if err == nil {
			inspectedBundle.Authentic = true
		}

		entity := a.Bundle
		verificationContent, err := entity.VerificationContent()
		if err != nil {
			return fmt.Errorf("failed to fetch verification content: %w", err)
		}

		if leafCert := verificationContent.GetCertificate(); leafCert != nil {

			certSummary, err := certificate.SummarizeCertificate(leafCert)
			if err != nil {
				return fmt.Errorf("failed to summarize certificate: %w", err)
			}

			inspectedBundle.Certificate = CertificateInspection{
				Summary:   certSummary,
				NotBefore: leafCert.NotBefore,
				NotAfter:  leafCert.NotAfter,
			}

		}

		sigContent, err := entity.SignatureContent()
		if err != nil {
			return fmt.Errorf("failed to fetch signature content: %w", err)
		}

		if envelope := sigContent.EnvelopeContent(); envelope != nil {
			stmt, err := envelope.Statement()
			if err != nil {
				return fmt.Errorf("failed to fetch envelope statement: %w", err)
			}

			inspectedBundle.Statement = *stmt
		}

		tlogTimestamps, err := dumpTlogs(entity)
		if err != nil {
			return fmt.Errorf("failed to dump tlog: %w", err)
		}
		inspectedBundle.TransparencyLogEntries = tlogTimestamps

		signedTimestamps, err := dumpSignedTimestamps(entity)
		if err != nil {
			return fmt.Errorf("failed to dump tsa: %w", err)
		}
		inspectedBundle.SignedTimestamps = signedTimestamps

		inspectedBundles = append(inspectedBundles, inspectedBundle)
	}

	inspectionResult := BundleInspectResult{InspectedBundles: inspectedBundles}

	// If the user provides the --format=json flag, print the results in JSON format
	if opts.exporter != nil {
		// print the results to the terminal as an array of JSON objects
		if err = opts.exporter.Write(opts.Logger.IO, inspectionResult); err != nil {
			return fmt.Errorf("failed to write JSON output")
		}
		return nil
	}

	printInspectionSummary(opts.Logger, inspectionResult.InspectedBundles)

	return nil
}

var logo = `
   _                       __ 
  (_)__  ___ ___  ___ ____/ /_
 / / _ \(_-</ _ \/ -_) __/ __/
/_/_//_/___/ .__/\__/\__/\__/ 
          /_/                 
`

func printInspectionSummary(logger *io.Handler, bundles []BundleInspection) {
	fmt.Printf("%s\n", logo)

	fmt.Printf("Found %s:\n---\n", text.Pluralize(len(bundles), "attestation"))

	bundleSummaries := make([][][]string, len(bundles))
	for i, iB := range bundles {
		bundleSummaries[i] = [][]string{
			[]string{"Authentic", formatAuthentic(iB.Authentic, iB.Certificate.CertificateIssuer)},
			[]string{"Source NWO", formatNwo(iB.Certificate.SourceRepositoryURI)},
			[]string{"PredicateType", iB.Statement.GetPredicateType()},
			[]string{"SubjectAlternativeName", iB.Certificate.SubjectAlternativeName},
			[]string{"RunInvocationURI", iB.Certificate.RunInvocationURI},
			[]string{"CertificateNotBefore", iB.Certificate.NotBefore.Format(time.RFC3339)},
		}
	}

	scheme := logger.ColorScheme
	for i, bundle := range bundleSummaries {
		for _, pair := range bundle {
			attr := fmt.Sprintf("%22s:", pair[0])
			fmt.Printf("%s %s\n", scheme.Bold(attr), pair[1])
		}
		if i < len(bundleSummaries)-1 {
			fmt.Println("---")
		}
	}
}

// formatNwo checks that longUrl is a valid URL and, if it is, extracts name/owner
// from "https://githubinstance.com/name/owner/file/path@refs/heads/foo"
func formatNwo(longUrl string) string {
	parsedUrl, err := url.Parse(longUrl)

	if err != nil {
		return longUrl
	}

	parts := strings.Split(parsedUrl.Path, "/")
	if len(parts) > 2 {
		return parts[1] + "/" + parts[2]
	} else {
		return parts[0]
	}
}

func formatAuthentic(authentic bool, certIssuer string) string {
	printIssuer := certIssuer

	if strings.HasSuffix(certIssuer, "O=GitHub\\, Inc.") {
		printIssuer = "(GH)"
	} else if strings.HasSuffix(certIssuer, "O=sigstore.dev") {
		printIssuer = "(PGI)"
	}

	return strconv.FormatBool(authentic) + " " + printIssuer
}

func dumpTlogs(entity *bundle.Bundle) ([]TlogEntryInspection, error) {
	inspectedTlogEntries := []TlogEntryInspection{}

	entries, err := entity.TlogEntries()
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		inspectedEntry := TlogEntryInspection{
			IntegratedTime: entry.IntegratedTime(),
			LogID:          entry.LogKeyID(),
		}

		inspectedTlogEntries = append(inspectedTlogEntries, inspectedEntry)
	}

	return inspectedTlogEntries, nil
}

func dumpSignedTimestamps(entity *bundle.Bundle) ([]time.Time, error) {
	timestamps := []time.Time{}

	signedTimestamps, err := entity.Timestamps()
	if err != nil {
		return nil, err
	}

	for _, signedTsBytes := range signedTimestamps {
		tsaTime, err := timestamp.ParseResponse(signedTsBytes)

		if err != nil {
			return nil, err
		}

		timestamps = append(timestamps, tsaTime.Time)
	}

	return timestamps, nil
}

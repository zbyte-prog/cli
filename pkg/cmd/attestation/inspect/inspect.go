package inspect

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/cli/cli/v2/internal/ghinstance"
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
	Verifiable             bool                  `json:"verifiable"`
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

		sigstoreRes := opts.SigstoreVerifier.Verify([]*api.Attestation{a}, sigstorePolicy)
		if sigstoreRes.Error == nil {
			inspectedBundle.Verifiable = true
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

			inspectedCert := CertificateInspection{
				Summary:   certSummary,
				NotBefore: leafCert.NotBefore,
				NotAfter:  leafCert.NotAfter,
			}

			inspectedBundle.Certificate = inspectedCert
			// PrettyPrint(inspectedCert)
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
			// PrettyPrint(stmt)
		}

		tlogTimestamps, err := dumpTlogs(entity)
		if err != nil {
			return fmt.Errorf("failed to dump tlog: %w", err)
		}
		inspectedBundle.TransparencyLogEntries = tlogTimestamps
		// PrettyPrint(tlogTimestamps)

		signedTimestamps, err := dumpSignedTimestamps(entity)
		if err != nil {
			return fmt.Errorf("failed to dump tsa: %w", err)
		}
		inspectedBundle.SignedTimestamps = signedTimestamps
		// PrettyPrint(signedTimestamps)

		// collect timestamps

		// fmt.Println(a.Bundle)
		inspectedBundles = append(inspectedBundles, inspectedBundle)
	}

	result := BundleInspectResult{
		InspectedBundles: inspectedBundles,
	}

	PrettyPrint(result)

	// policy, err := buildPolicy(*artifact)
	// if err != nil {
	// 	return fmt.Errorf("failed to build policy: %v", err)
	// }
	//
	// res := opts.SigstoreVerifier.Verify(attestations, policy)
	// if res.Error != nil {
	// 	return fmt.Errorf("at least one attestation failed to verify against Sigstore: %v", res.Error)
	// }
	//
	// opts.Logger.VerbosePrint(opts.Logger.ColorScheme.Green(
	// 	"Successfully verified all attestations against Sigstore!\n\n",
	// ))
	//
	// If the user provides the --format=json flag, print the results in JSON format
	// if opts.exporter != nil {
	// 	details, err := getAttestationDetails(opts.Tenant, res.VerifyResults)
	// 	if err != nil {
	// 		return fmt.Errorf("failed to get attestation detail: %v", err)
	// 	}
	//
	// 	// print the results to the terminal as an array of JSON objects
	// 	if err = opts.exporter.Write(opts.Logger.IO, details); err != nil {
	// 		return fmt.Errorf("failed to write JSON output")
	// 	}
	// 	return nil
	// }
	//
	// // otherwise, print results in a table
	// details, err := getDetailsAsSlice(opts.Tenant, res.VerifyResults)
	// if err != nil {
	// 	return fmt.Errorf("failed to parse attestation details: %v", err)
	// }
	//
	// headers := []string{"Repo Name", "Repo ID", "Org Name", "Org ID", "Workflow ID"}
	// t := tableprinter.New(opts.Logger.IO, tableprinter.WithHeader(headers...))
	//
	// for _, row := range details {
	// 	for _, field := range row {
	// 		t.AddField(field, tableprinter.WithTruncate(nil))
	// 	}
	// 	t.EndRow()
	// }
	//
	// if err = t.Render(); err != nil {
	// 	return fmt.Errorf("failed to print output: %v", err)
	// }
	//

	return nil
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

func PrettyPrint(v interface{}) (err error) {
	b, err := json.MarshalIndent(v, "", "  ")

	if err == nil {
		fmt.Println(string(b))
		return nil
	}
	return err
}

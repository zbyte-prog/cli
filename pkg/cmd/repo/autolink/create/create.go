package create

import (
	"fmt"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/repo/autolink/domain"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type createOptions struct {
	BaseRepo       func() (ghrepo.Interface, error)
	Browser        browser.Browser
	AutolinkClient AutolinkCreateClient
	IO             *iostreams.IOStreams
	Exporter       cmdutil.Exporter

	KeyPrefix   string
	URLTemplate string
	Numeric     bool
}

type AutolinkCreateClient interface {
	Create(repo ghrepo.Interface, request AutolinkCreateRequest) (*domain.Autolink, error)
}

func NewCmdCreate(f *cmdutil.Factory, runF func(*createOptions) error) *cobra.Command {
	opts := &createOptions{
		Browser: f.Browser,
		IO:      f.IOStreams,
	}

	cmd := &cobra.Command{
		Use:   "create <keyPrefix> <urlTemplate>",
		Short: "Create a new autolink reference",
		Long: heredoc.Docf(`
			Create a new autolink reference for a repository.

			Autolinks automatically generate links to external resources when they appear in an issue, pull request, or commit.

			The %[1]skeyPrefix%[1]s specifies the prefix that will generate a link when it is appended by certain characters.

			The %[1]surlTemplate%[1]s specifies the target URL that will be generated when the keyPrefix is found. The urlTemplate must contain %[1]s<num>%[1]s for the reference number. %[1]s<num>%[1]s matches different characters depending on the whether the autolink is specified as numeric or alphanumeric.

			By default, the command will create an alphanumeric autolink. This means that  the %[1]s<num>%[1]s in the %[1]surlTemplate%[1]s will match alphanumeric characters %[1]sA-Z%[1]s (case insensitive), %[1]s0-9%[1]s, and %[1]s-%[1]s. To create a numeric autolink, use the %[1]s--numeric%[1]s flag. Numeric autolinks only match against numeric characters. If the template contains multiple instances of %[1]s<num>%[1]s, only the first will be replaced.

			If you are using a shell that applies special meaning to angle brackets, you you will need to need to escape these characters in the %[1]surlTemplate%[1]s or place quotation marks around this argument.
			
			Only repository administrators can create autolinks.
		`, "`"),
		Example: heredoc.Doc(`
			# Create an alphanumeric autolink to example.com for the key prefix "TICKET-". This will generate a link to https://example.com/TICKET?query=123abc when "TICKET-123abc" appears.
			$ gh repo autolink create TICKET- https://example.com/TICKET?query=<num>

			# Create a numeric autolink to example.com for the key prefix "STORY-". This will generate a link to https://example.com/STORY?id=123 when "STORY-123" appears.
			$ gh repo autolink create STORY- https://example.com/STORY?id=<num> --numeric

		`),
		Args:    cmdutil.ExactArgs(2, "Cannot create autolink: keyPrefix and urlTemplate arguments are both required"),
		Aliases: []string{"new"},
		RunE: func(c *cobra.Command, args []string) error {
			opts.BaseRepo = f.BaseRepo
			httpClient, err := f.HttpClient()

			if err != nil {
				return err
			}

			opts.AutolinkClient = &AutolinkCreator{HTTPClient: httpClient}
			opts.KeyPrefix = args[0]
			opts.URLTemplate = args[1]

			if runF != nil {
				return runF(opts)
			}

			return createRun(opts)
		},
	}

	cmd.Flags().BoolVarP(&opts.Numeric, "numeric", "n", false, "Mark autolink as non-alphanumeric")

	return cmd
}

func createRun(opts *createOptions) error {
	repo, err := opts.BaseRepo()
	if err != nil {
		return err
	}
	cs := opts.IO.ColorScheme()

	isAlphanumeric := !opts.Numeric

	request := AutolinkCreateRequest{
		KeyPrefix:      opts.KeyPrefix,
		URLTemplate:    opts.URLTemplate,
		IsAlphanumeric: isAlphanumeric,
	}

	autolink, err := opts.AutolinkClient.Create(repo, request)

	if err != nil {
		return fmt.Errorf("%s %w", cs.Red("error creating autolink:"), err)
	}

	msg := successMsg(autolink, repo, cs)
	fmt.Fprint(opts.IO.Out, msg)

	return nil
}

func successMsg(autolink *domain.Autolink, repo ghrepo.Interface, cs *iostreams.ColorScheme) string {
	autolinkType := "Numeric"
	if autolink.IsAlphanumeric {
		autolinkType = "Alphanumeric"
	}

	createdMsg := fmt.Sprintf(
		"%s %s autolink created in %s (id: %d)",
		cs.SuccessIconWithColor(cs.Green),
		autolinkType,
		ghrepo.FullName(repo),
		autolink.ID,
	)

	autolinkMapMsg := cs.Bluef(
		"  %s%s → %s",
		autolink.KeyPrefix,
		"<num>",
		autolink.URLTemplate,
	)

	return fmt.Sprintf("%s\n%s\n", createdMsg, autolinkMapMsg)
}

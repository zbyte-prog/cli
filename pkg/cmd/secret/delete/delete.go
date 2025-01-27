package delete

import (
	"fmt"
	"net/http"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/api"
	"github.com/cli/cli/v2/internal/gh"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmd/secret/shared"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type DeleteOptions struct {
	HttpClient func() (*http.Client, error)
	IO         *iostreams.IOStreams
	Config     func() (gh.Config, error)
	BaseRepo   func() (ghrepo.Interface, error)

	SecretName  string
	OrgName     string
	EnvName     string
	UserSecrets bool
	Application string
}

func NewCmdDelete(f *cmdutil.Factory, runF func(*DeleteOptions) error) *cobra.Command {
	opts := &DeleteOptions{
		IO:         f.IOStreams,
		Config:     f.Config,
		HttpClient: f.HttpClient,
	}

	cmd := &cobra.Command{
		Use:   "delete <secret-name>",
		Short: "Delete secrets",
		Long: heredoc.Doc(`
			Delete a secret on one of the following levels:
			- repository (default): available to GitHub Actions runs or Dependabot in a repository
			- environment: available to GitHub Actions runs for a deployment environment in a repository
			- organization: available to GitHub Actions runs, Dependabot, or Codespaces within an organization
			- user: available to Codespaces for your user
		`),
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// If the user specified a repo directly, then we're using the OverrideBaseRepoFunc set by EnableRepoOverride
			// So there's no reason to use the specialised BaseRepoFunc that requires remote disambiguation.
			opts.BaseRepo = f.BaseRepo
			if !cmd.Flags().Changed("repo") {
				// If they haven't specified a repo directly, then we will wrap the BaseRepoFunc in one that errors if
				// there might be multiple valid remotes.
				opts.BaseRepo = shared.RequireNoAmbiguityBaseRepoFunc(opts.BaseRepo, f.Remotes)
				// But if we are able to prompt, then we will wrap that up in a BaseRepoFunc that can prompt the user to
				// resolve the ambiguity.
				if opts.IO.CanPrompt() {
					opts.BaseRepo = shared.PromptWhenAmbiguousBaseRepoFunc(opts.BaseRepo, f.IOStreams, f.Prompter)
				}
			}

			if err := cmdutil.MutuallyExclusive("specify only one of `--org`, `--env`, or `--user`", opts.OrgName != "", opts.EnvName != "", opts.UserSecrets); err != nil {
				return err
			}

			opts.SecretName = args[0]

			if runF != nil {
				return runF(opts)
			}

			return removeRun(opts)
		},
		Aliases: []string{
			"remove",
		},
	}
	cmd.Flags().StringVarP(&opts.OrgName, "org", "o", "", "Delete a secret for an organization")
	cmd.Flags().StringVarP(&opts.EnvName, "env", "e", "", "Delete a secret for an environment")
	cmd.Flags().BoolVarP(&opts.UserSecrets, "user", "u", false, "Delete a secret for your user")
	cmdutil.StringEnumFlag(cmd, &opts.Application, "app", "a", "", []string{shared.Actions, shared.Codespaces, shared.Dependabot}, "Delete a secret for a specific application")

	return cmd
}

func removeRun(opts *DeleteOptions) error {
	c, err := opts.HttpClient()
	if err != nil {
		return fmt.Errorf("could not create http client: %w", err)
	}
	client := api.NewClientFromHTTP(c)

	orgName := opts.OrgName
	envName := opts.EnvName

	secretEntity, err := shared.GetSecretEntity(orgName, envName, opts.UserSecrets)
	if err != nil {
		return err
	}

	var baseRepo ghrepo.Interface
	if secretEntity == shared.Repository || secretEntity == shared.Environment {
		baseRepo, err = opts.BaseRepo()
		if err != nil {
			return err
		}
	}

	secretApp, err := shared.GetSecretApp(opts.Application, secretEntity)
	if err != nil {
		return err
	}

	if !shared.IsSupportedSecretEntity(secretApp, secretEntity) {
		return fmt.Errorf("%s secrets are not supported for %s", secretEntity, secretApp)
	}

	cfg, err := opts.Config()
	if err != nil {
		return err
	}

	var path string
	var host string
	switch secretEntity {
	case shared.Organization:
		path = fmt.Sprintf("orgs/%s/%s/secrets/%s", orgName, secretApp, opts.SecretName)
		host, _ = cfg.Authentication().DefaultHost()
	case shared.Environment:
		path = fmt.Sprintf("repos/%s/environments/%s/secrets/%s", ghrepo.FullName(baseRepo), envName, opts.SecretName)
		host = baseRepo.RepoHost()
	case shared.User:
		path = fmt.Sprintf("user/codespaces/secrets/%s", opts.SecretName)
		host, _ = cfg.Authentication().DefaultHost()
	case shared.Repository:
		path = fmt.Sprintf("repos/%s/%s/secrets/%s", ghrepo.FullName(baseRepo), secretApp, opts.SecretName)
		host = baseRepo.RepoHost()
	}

	err = client.REST(host, "DELETE", path, nil, nil)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", opts.SecretName, err)
	}

	if opts.IO.IsStdoutTTY() {
		var target string
		switch secretEntity {
		case shared.Organization:
			target = orgName
		case shared.User:
			target = "your user"
		case shared.Repository, shared.Environment:
			target = ghrepo.FullName(baseRepo)
		}

		cs := opts.IO.ColorScheme()
		if envName != "" {
			fmt.Fprintf(opts.IO.Out, "%s Deleted secret %s from %s environment on %s\n", cs.SuccessIconWithColor(cs.Red), opts.SecretName, envName, target)
		} else {
			fmt.Fprintf(opts.IO.Out, "%s Deleted %s secret %s from %s\n", cs.SuccessIconWithColor(cs.Red), secretApp.Title(), opts.SecretName, target)
		}
	}

	return nil
}

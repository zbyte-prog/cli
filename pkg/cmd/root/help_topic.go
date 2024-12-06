package root

import (
	"fmt"
	"io"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/text"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/spf13/cobra"
)

type helpTopic struct {
	name    string
	short   string
	long    string
	example string
}

var HelpTopics = []helpTopic{
	{
		name:  "mintty",
		short: "Information about using gh with MinTTY",
		long: heredoc.Docf(`
			MinTTY is the terminal emulator that comes by default with Git
			for Windows. It has known issues with gh's ability to prompt a
			user for input.

			There are a few workarounds to make gh work with MinTTY:

			- Reinstall Git for Windows, checking "Enable experimental support for pseudo consoles".

			- Use a different terminal emulator with Git for Windows like Windows Terminal.
			  You can run %[1]sC:\Program Files\Git\bin\bash.exe%[1]s from any terminal emulator to continue
			  using all of the tooling in Git For Windows without MinTTY.

			- Prefix invocations of gh with %[1]swinpty%[1]s, eg: %[1]swinpty gh auth login%[1]s.
			  NOTE: this can lead to some UI bugs.
		`, "`"),
	},
	{
		name:  "environment",
		short: "Environment variables that can be used with gh",
		long: heredoc.Docf(`	
			%[1]sGH_TOKEN%[1]s, %[1]sGITHUB_TOKEN%[1]s (in order of precedence): an authentication token that will be used when
			a command targets either github.com or a subdomain of ghe.com. Setting this avoids being prompted to
			authenticate and takes precedence over previously stored credentials.

			%[1]sGH_ENTERPRISE_TOKEN%[1]s, %[1]sGITHUB_ENTERPRISE_TOKEN%[1]s (in order of precedence): an authentication 
			token that will be used when a command targets a GitHub Enterprise Server host.

			%[1]sGH_HOST%[1]s: specify the GitHub hostname for commands where a hostname has not been provided, or
			cannot be inferred from the context of a local Git repository. If this host was previously
			authenticated with, the stored credentials will be used. Otherwise, setting %[1]sGH_TOKEN%[1]s or
			%[1]sGH_ENTERPRISE_TOKEN%[1]s is required, depending on the targeted host.

			%[1]sGH_REPO%[1]s: specify the GitHub repository in the %[1]s[HOST/]OWNER/REPO%[1]s format for commands
			that otherwise operate on a local repository.

			%[1]sGH_EDITOR%[1]s, %[1]sGIT_EDITOR%[1]s, %[1]sVISUAL%[1]s, %[1]sEDITOR%[1]s (in order of precedence): the editor tool to use
			for authoring text.

			%[1]sGH_BROWSER%[1]s, %[1]sBROWSER%[1]s (in order of precedence): the web browser to use for opening links.

			%[1]sGH_DEBUG%[1]s: set to a truthy value to enable verbose output on standard error. Set to %[1]sapi%[1]s
			to additionally log details of HTTP traffic.

			%[1]sDEBUG%[1]s (deprecated): set to %[1]s1%[1]s, %[1]strue%[1]s, or %[1]syes%[1]s to enable verbose output on standard
			error.

			%[1]sGH_PAGER%[1]s, %[1]sPAGER%[1]s (in order of precedence): a terminal paging program to send standard output
			to, e.g. %[1]sless%[1]s.

			%[1]sGLAMOUR_STYLE%[1]s: the style to use for rendering Markdown. See
			<https://github.com/charmbracelet/glamour#styles>

			%[1]sNO_COLOR%[1]s: set to any value to avoid printing ANSI escape sequences for color output.

			%[1]sCLICOLOR%[1]s: set to %[1]s0%[1]s to disable printing ANSI colors in output.

			%[1]sCLICOLOR_FORCE%[1]s: set to a value other than %[1]s0%[1]s to keep ANSI colors in output
			even when the output is piped.

			%[1]sGH_FORCE_TTY%[1]s: set to any value to force terminal-style output even when the output is
			redirected. When the value is a number, it is interpreted as the number of columns
			available in the viewport. When the value is a percentage, it will be applied against
			the number of columns available in the current viewport.

			%[1]sGH_NO_UPDATE_NOTIFIER%[1]s: set to any value to disable update notifications. By default, gh
			checks for new releases once every 24 hours and displays an upgrade notice on standard
			error if a newer version was found.

			%[1]sGH_CONFIG_DIR%[1]s: the directory where gh will store configuration files. If not specified, 
			the default value will be one of the following paths (in order of precedence):
			  - %[1]s$XDG_CONFIG_HOME/gh%[1]s (if %[1]s$XDG_CONFIG_HOME%[1]s is set),
			  - %[1]s$AppData/GitHub CLI%[1]s (on Windows if %[1]s$AppData%[1]s is set), or
			  - %[1]s$HOME/.config/gh%[1]s.

			%[1]sGH_PROMPT_DISABLED%[1]s: set to any value to disable interactive prompting in the terminal.

			%[1]sGH_PATH%[1]s: set the path to the gh executable, useful for when gh can not properly determine
			its own path such as in the cygwin terminal.

			%[1]sGH_MDWIDTH%[1]s: default maximum width for markdown render wrapping.  The max width of lines
			wrapped on the terminal will be taken as the lesser of the terminal width, this value, or 120 if
			not specified.  This value is used, for example, with %[1]spr view%[1]s subcommand.
		`, "`"),
	},
	{
		name:  "reference",
		short: "A comprehensive reference of all gh commands",
	},
	{
		name:  "formatting",
		short: "Formatting options for JSON data exported from gh",
		long: heredoc.Docf(`
			By default, the result of %[1]sgh%[1]s commands are output in line-based plain text format.
			Some commands support passing the %[1]s--json%[1]s flag, which converts the output to JSON format.
			Once in JSON, the output can be further formatted according to a required formatting string by
			adding either the %[1]s--jq%[1]s or %[1]s--template%[1]s flag. This is useful for selecting a subset of data,
			creating new data structures, displaying the data in a different format, or as input to another
			command line script.

			The %[1]s--json%[1]s flag requires a comma separated list of fields to fetch. To view the possible JSON
			field names for a command omit the string argument to the %[1]s--json%[1]s flag when you run the command.
			Note that you must pass the %[1]s--json%[1]s flag and field names to use the %[1]s--jq%[1]s or %[1]s--template%[1]s flags.

			The %[1]s--jq%[1]s flag requires a string argument in jq query syntax, and will only print
			those JSON values which match the query. jq queries can be used to select elements from an
			array, fields from an object, create a new array, and more. The %[1]sjq%[1]s utility does not need
			to be installed on the system to use this formatting directive. When connected to a terminal,
			the output is automatically pretty-printed. To learn about jq query syntax, see:
			<https://jqlang.github.io/jq/manual/>

			The %[1]s--template%[1]s flag requires a string argument in Go template syntax, and will only print
			those JSON values which match the query.
			In addition to the Go template functions in the standard library, the following functions can be used
			with this formatting directive:
			- %[1]sautocolor%[1]s: like %[1]scolor%[1]s, but only emits color to terminals
			- %[1]scolor <style> <input>%[1]s: colorize input using <https://github.com/mgutz/ansi>
			- %[1]sjoin <sep> <list>%[1]s: joins values in the list using a separator
			- %[1]spluck <field> <list>%[1]s: collects values of a field from all items in the input
			- %[1]stablerow <fields>...%[1]s: aligns fields in output vertically as a table
			- %[1]stablerender%[1]s: renders fields added by tablerow in place
			- %[1]stimeago <time>%[1]s: renders a timestamp as relative to now
			- %[1]stimefmt <format> <time>%[1]s: formats a timestamp using Go's %[1]sTime.Format%[1]s function
			- %[1]struncate <length> <input>%[1]s: ensures input fits within length
			- %[1]shyperlink <url> <text>%[1]s: renders a terminal hyperlink

			To learn more about Go templates, see: <https://golang.org/pkg/text/template/>.
		`, "`"),
		example: heredoc.Doc(`
			# default output format
			$ gh pr list
			Showing 23 of 23 open pull requests in cli/cli

			#123  A helpful contribution          contribution-branch              about 1 day ago
			#124  Improve the docs                docs-branch                      about 2 days ago
			#125  An exciting new feature         feature-branch                   about 2 days ago


			# adding the --json flag with a list of field names
			$ gh pr list --json number,title,author
			[
			  {
			    "author": {
			      "login": "monalisa"
			    },
			    "number": 123,
			    "title": "A helpful contribution"
			  },
			  {
			    "author": {
			      "login": "codercat"
			    },
			    "number": 124,
			    "title": "Improve the docs"
			  },
			  {
			    "author": {
			      "login": "cli-maintainer"
			    },
			    "number": 125,
			    "title": "An exciting new feature"
			  }
			]


			# adding the --jq flag and selecting fields from the array
			$ gh pr list --json author --jq '.[].author.login'
			monalisa
			codercat
			cli-maintainer

			# --jq can be used to implement more complex filtering and output changes:
			$ gh issue list --json number,title,labels --jq \
			  'map(select((.labels | length) > 0))    # must have labels
			  | map(.labels = (.labels | map(.name))) # show only the label names
			  | .[:3]                                 # select the first 3 results'
			  [
			    {
			      "labels": [
			        "enhancement",
			        "needs triage"
			      ],
			      "number": 123,
			      "title": "A helpful contribution"
			    },
			    {
			      "labels": [
			        "help wanted",
			        "docs",
			        "good first issue"
			      ],
			      "number": 125,
			      "title": "Improve the docs"
			    },
			    {
			      "labels": [
			        "enhancement",
			      ],
			      "number": 7221,
			      "title": "An exciting new feature"
			    }
			  ]
			# using the --template flag with the hyperlink helper
			gh issue list --json title,url --template '{{range .}}{{hyperlink .url .title}}{{"\n"}}{{end}}'


			# adding the --template flag and modifying the display format
			$ gh pr list --json number,title,headRefName,updatedAt --template \
				'{{range .}}{{tablerow (printf "#%v" .number | autocolor "green") .title .headRefName (timeago .updatedAt)}}{{end}}'

			#123  A helpful contribution      contribution-branch       about 1 day ago
			#124  Improve the docs            docs-branch               about 2 days ago
			#125  An exciting new feature     feature-branch            about 2 days ago


			# a more complex example with the --template flag which formats a pull request using multiple tables with headers:
			$ gh pr view 3519 --json number,title,body,reviews,assignees --template \
			'{{printf "#%v" .number}} {{.title}}

			{{.body}}

			{{tablerow "ASSIGNEE" "NAME"}}{{range .assignees}}{{tablerow .login .name}}{{end}}{{tablerender}}
			{{tablerow "REVIEWER" "STATE" "COMMENT"}}{{range .reviews}}{{tablerow .author.login .state .body}}{{end}}
			'

			#3519 Add table and helper template functions

			Resolves #3488

			ASSIGNEE  NAME
			mislav    Mislav Marohnić


			REVIEWER  STATE              COMMENT
			mislav    COMMENTED          This is going along great! Thanks for working on this ❤️
		`),
	},
	{
		name:  "exit-codes",
		short: "Exit codes used by gh",
		long: heredoc.Doc(`
			gh follows normal conventions regarding exit codes.

			- If a command completes successfully, the exit code will be 0

			- If a command fails for any reason, the exit code will be 1

			- If a command is running but gets cancelled, the exit code will be 2

			- If a command requires authentication, the exit code will be 4

			NOTE: It is possible that a particular command may have more exit codes, so it is a good
			practice to check documentation for the command if you are relying on exit codes to
			control some behavior.
		`),
	},
	{
		name:  "scopes",
		short: "Auth scopes for token used by gh",
		long: heredoc.Docf(`
			Scopes let you specify exactly what type of access you need. Scopes limit access for OAuth tokens.
			They do not grant any additional permission beyond that which the user already has.
			- %[1]s(no scope)%[1]s: grants read-only access to public information (including user profile info, repository 	info, and gists).
			- %[1]srepo%[1]s: grants full access to public and private repositories including read and write access to code,
			commit statuses, repository invitations, collaborators, deployment statuses, and repository webhooks.
			  NOTE: In addition to repository related resources, the repo scope also grants access to manage organization-owned
			  resources including projects, invitations, team memberships and webhooks.
			  This scope also grants the ability to manage projects owned by users.
			  - %[1]srepo:status%[1]s: grants read/write access to commit statuses in public and private repositories.
			This scope is only necessary to grant other users or services access to private repository commit statuses without granting access to the code.
			  - %[1]srepo_deployment%[1]s: grants access to deployment statuses for public and private repositories.
			This scope is only necessary to grant other users or services access to deployment statuses, without granting access to the code.
			  - %[1]spublic_repo%[1]s: limits access to public repositories. That includes read/write access to code,
			commit statuses, repository projects, collaborators, and deployment statuses for public repositories and
			organizations. Also required for starring public repositories.
			  - %[1]srepo:invite%[1]s: grants accept/decline abilities for invitations to collaborate on a repository.
			This scope is only necessary to grant other users or services access to invites without granting access to the code.
			  - %[1]ssecurity_events%[1]s: grants read and write access to security events in the code scanning API.
			This scope is only necessary to grant other users or services access to security events without granting access to the code.
			- %[1]sadmin:repo_hook%[1]s: grants read, write, ping, and delete access to repository hooks in public or private repositories.
			The %[1]srepo%[1]s and %[1]spublic_repo%[1]s scopes grant full access to repositories, including repository hooks. Use the %[1]sadmin:repo_hook%[1]s scope to limit access to only repository hooks.
			  - %[1]swrite:repo_hook%[1]s: grants read, write, and ping access to hooks in public or private repositories.
			  - %[1]sread:repo_hook%[1]s: grants read and ping access to hooks in public or private repositories.
			- %[1]sadmin:org%[1]s: fully manage the organization and its teams, projects, and memberships.
			  - %[1]swrite:org%[1]s: read and write access to organization membership and organization projects.
			  - %[1]sread:org%[1]s: read-only access to organization membership, organization projects, and team membership.
			- %[1]sadmin:public_key%[1]s: fully manage public keys.
			  - %[1]swrite:public_key%[1]s: create, list, and view details for public keys.
			  - %[1]sread:public_key%[1]s: list and view details for public keys.
			- %[1]sadmin:org_hook%[1]s: grants read, write, ping, and delete access to organization hooks.
			  NOTE: OAuth tokens will only be able to perform these actions on organization hooks which were created by the OAuth app.
			  Personal access tokens will only be able to perform these actions on organization hooks created by a user.
			- %[1]sgist%[1]s: grants write access to gists.
			- %[1]snotifications%[1]s: grants:
			  - read access to a user's notifications
			  - mark as read access to threads
			  - watch and unwatch access to a repository, and
			  - read, write, and delete access to thread subscriptions.
			- %[1]suser%[1]s: grants read/write access to profile info only. Note that this scope includes %[1]suser:email%[1]s and %[1]suser:follow%[1]s.
			  - %[1]sread:user%[1]s: grants access to read a user's profile data.
			  - %[1]suser:email%[1]s: grants read access to a user's email addresses.
			  - %[1]suser:follow%[1]s: grants access to follow or unfollow other users.
			- %[1]sproject%[1]s: grants read/write access to user and organization projects.
			  - %[1]sread:project%[1]s: grants read only access to user and organization projects.
			- %[1]sdelete_repo%[1]s: grants access to delete adminable repositories.
			- %[1]swrite:packages%[1]s: grants access to upload or publish a package in GitHub Packages.
			- %[1]sread:packages%[1]s: grants access to download or install packages from GitHub Packages.
			- %[1]sdelete:packages%[1]s: grants access to delete packages from GitHub Packages.
			- %[1]sadmin:gpg_key%[1]s: fully manage GPG keys.
			  - %[1]swrite:gpg_key%[1]s: create, list, and view details for GPG keys.
			  - %[1]sread:gpg_key%[1]s: list and view details for GPG keys.
			- %[1]scodespace%[1]s: grants the ability to create and manage codespaces. Codespaces can expose a
			%[1]sGITHUB_TOKEN%[1]s which may have a different set of scopes.
			- %[1]sworkflow%[1]s: grants the ability to add and update GitHub Actions workflow files.
			Workflow files can be committed without this scope if the same file (with both the same path and contents) exists
			on another branch in the same repository. Workflow files can expose %[1]sGITHUB_TOKEN%[1]s which may have a
			different set of scopes.
			NOTE: You can always find up-to-date list of scopes at <https://docs.github.com/en/developers/apps/building-oauth-apps/scopes-for-oauth-apps/#available-scopes>.
		`, "`"),
	},
}

func NewCmdHelpTopic(ios *iostreams.IOStreams, ht helpTopic) *cobra.Command {
	cmd := &cobra.Command{
		Use:     ht.name,
		Short:   ht.short,
		Long:    ht.long,
		Example: ht.example,
		Hidden:  true,
		Annotations: map[string]string{
			"markdown:generate": "true",
			"markdown:basename": "gh_help_" + ht.name,
		},
	}

	cmd.SetUsageFunc(func(c *cobra.Command) error {
		return helpTopicUsageFunc(ios.ErrOut, c)
	})

	cmd.SetHelpFunc(func(c *cobra.Command, _ []string) {
		helpTopicHelpFunc(ios.Out, c)
	})

	return cmd
}

func helpTopicHelpFunc(w io.Writer, command *cobra.Command) {
	fmt.Fprint(w, command.Long)
	if command.Example != "" {
		fmt.Fprintf(w, "\n\nEXAMPLES\n")
		fmt.Fprint(w, text.Indent(command.Example, "  "))
	}
}

func helpTopicUsageFunc(w io.Writer, command *cobra.Command) error {
	fmt.Fprintf(w, "Usage: gh help %s", command.Use)
	return nil
}

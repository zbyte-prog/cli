package autolink

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/browser"
	"github.com/cli/cli/v2/internal/ghrepo"
	"github.com/cli/cli/v2/pkg/cmdutil"
	"github.com/cli/cli/v2/pkg/httpmock"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/google/shlex"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdList(t *testing.T) {
	tests := []struct {
		name         string
		input        string
		output       listOptions
		wantErr      bool
		wantExporter bool
		errMsg       string
	}{
		{
			name:   "no argument",
			input:  "",
			output: listOptions{},
		},
		{
			name:   "web flag",
			input:  "--web",
			output: listOptions{WebMode: true},
		},
		{
			name:         "json flag",
			input:        "--json id",
			output:       listOptions{},
			wantExporter: true,
		},
		{
			name:    "invalid json flag",
			input:   "--json invalid",
			wantErr: true,
			errMsg: heredoc.Doc(`
				Unknown JSON field: "invalid"
				Available fields:
				  id
				  isAlphanumeric
				  keyPrefix
				  urlTemplate`),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, _, _ := iostreams.Test()
			f := &cmdutil.Factory{
				IOStreams: ios,
			}

			argv, err := shlex.Split(tt.input)
			require.NoError(t, err)

			var gotOpts *listOptions
			cmd := newCmdList(f, func(opts *listOptions) error {
				gotOpts = opts
				return nil
			})

			cmd.SetArgs(argv)
			cmd.SetIn(&bytes.Buffer{})
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})

			_, err = cmd.ExecuteC()
			if tt.wantErr {
				require.EqualError(t, err, tt.errMsg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.output.WebMode, gotOpts.WebMode)
			assert.Equal(t, tt.wantExporter, gotOpts.Exporter != nil)
		})
	}
}

func TestListRun(t *testing.T) {
	tests := []struct {
		name       string
		opts       *listOptions
		isTTY      bool
		httpStubs  func(t *testing.T, reg *httpmock.Registry)
		wantStdout string
		wantStderr string
		wantErr    bool
	}{
		{
			name:  "list tty",
			opts:  &listOptions{},
			isTTY: true,
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/autolinks"),
					httpmock.StringResponse(`[
						{
							"id": 1,
							"key_prefix": "TICKET-",
							"url_template": "https://example.com/TICKET?query=<num>",
							"is_alphanumeric": true
						},
						{
							"id": 2,
							"key_prefix": "STORY-",
							"url_template": "https://example.com/STORY?id=<num>",
							"is_alphanumeric": false
						}
					]`),
				)
			},
			wantStdout: heredoc.Doc(`

				Showing 2 autolink references in OWNER/REPO

				ID  KEY PREFIX  URL TEMPLATE                            ALPHANUMERIC
				1   TICKET-     https://example.com/TICKET?query=<num>  true
				2   STORY-      https://example.com/STORY?id=<num>      false
			`),
			wantStderr: "",
		},
		{
			name: "list json",
			opts: &listOptions{
				Exporter: func() cmdutil.Exporter {
					exporter := cmdutil.NewJSONExporter()
					exporter.SetFields([]string{"id"})
					return exporter
				}(),
			},
			isTTY: true,
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/autolinks"),
					httpmock.StringResponse(`[
						{
							"id": 1,
							"key_prefix": "TICKET-",
							"url_template": "https://example.com/TICKET?query=<num>",
							"is_alphanumeric": true
						},
						{
							"id": 2,
							"key_prefix": "STORY-",
							"url_template": "https://example.com/STORY?id=<num>",
							"is_alphanumeric": false
						}
					]`),
				)
			},
			wantStdout: "[{\"id\":1},{\"id\":2}]\n",
			wantStderr: "",
		},
		{
			name:  "list non-tty",
			opts:  &listOptions{},
			isTTY: false,
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/autolinks"),
					httpmock.StringResponse(`[
						{
							"id": 1,
							"key_prefix": "TICKET-",
							"url_template": "https://example.com/TICKET?query=<num>",
							"is_alphanumeric": true
						},
						{
							"id": 2,
							"key_prefix": "STORY-",
							"url_template": "https://example.com/STORY?id=<num>",
							"is_alphanumeric": false
						}
					]`),
				)
			},
			wantStdout: heredoc.Doc(`
				1	TICKET-	https://example.com/TICKET?query=<num>	true
				2	STORY-	https://example.com/STORY?id=<num>	false
			`),
			wantStderr: "",
		},

		{
			name:  "no results",
			opts:  &listOptions{},
			isTTY: true,
			httpStubs: func(t *testing.T, reg *httpmock.Registry) {
				reg.Register(
					httpmock.REST("GET", "repos/OWNER/REPO/autolinks"),
					httpmock.StringResponse(`[]`),
				)
			},
			wantStdout: "",
			wantStderr: "",
			wantErr:    true,
		},
		{
			name:       "web mode",
			isTTY:      true,
			opts:       &listOptions{WebMode: true},
			wantStderr: "Opening https://github.com/OWNER/REPO/settings/key_links in your browser.\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ios, _, stdout, stderr := iostreams.Test()
			ios.SetStdoutTTY(tt.isTTY)
			ios.SetStdinTTY(tt.isTTY)
			ios.SetStderrTTY(tt.isTTY)

			reg := &httpmock.Registry{}
			if tt.httpStubs != nil {
				tt.httpStubs(t, reg)
			}

			defer reg.Verify(t)

			opts := tt.opts
			opts.IO = ios
			opts.Browser = &browser.Stub{}

			opts.IO = ios
			opts.BaseRepo = func() (ghrepo.Interface, error) { return ghrepo.New("OWNER", "REPO"), nil }

			opts.HTTPClient = func() (*http.Client, error) { return &http.Client{Transport: reg}, nil }

			err := listRun(opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("listRun() return error: %v", err)
				return
			}

			if stdout.String() != tt.wantStdout {
				t.Errorf("wants stdout %q, got %q", tt.wantStdout, stdout.String())
			}
			if stderr.String() != tt.wantStderr {
				t.Errorf("wants stderr %q, got %q", tt.wantStderr, stderr.String())
			}
		})
	}
}

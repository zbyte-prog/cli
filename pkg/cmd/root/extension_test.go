package root_test

import (
	"io"
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/cli/cli/v2/internal/update"
	"github.com/cli/cli/v2/pkg/cmd/root"
	"github.com/cli/cli/v2/pkg/extensions"
	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCmdExtension_Updates(t *testing.T) {
	tests := []struct {
		name               string
		extCurrentVersion  string
		extIsPinned        bool
		extLatestVersion   string
		extName            string
		extUpdateAvailable bool
		extURL             string
		wantStderr         string
	}{
		{
			name:               "no update available",
			extName:            "no-update",
			extUpdateAvailable: false,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "1.0.0",
			extURL:             "https//github.com/dne/no-update",
		},
		{
			name:               "major update",
			extName:            "major-update",
			extUpdateAvailable: true,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "2.0.0",
			extURL:             "https//github.com/dne/major-update",
			wantStderr: heredoc.Doc(`
				A new release of major-update is available: 1.0.0 → 2.0.0
				To upgrade, run: gh extension upgrade major-update
				https//github.com/dne/major-update
			`),
		},
		{
			name:               "major update, pinned",
			extName:            "major-update-pin",
			extUpdateAvailable: true,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "2.0.0",
			extIsPinned:        true,
			extURL:             "https//github.com/dne/major-update",
			wantStderr: heredoc.Doc(`
				A new release of major-update-pin is available: 1.0.0 → 2.0.0
				To upgrade, run: gh extension upgrade major-update-pin --force
				https//github.com/dne/major-update
			`),
		},
		{
			name:               "minor update",
			extName:            "minor-update",
			extUpdateAvailable: true,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "1.1.0",
			extURL:             "https//github.com/dne/minor-update",
			wantStderr: heredoc.Doc(`
				A new release of minor-update is available: 1.0.0 → 1.1.0
				To upgrade, run: gh extension upgrade minor-update
				https//github.com/dne/minor-update
			`),
		},
		{
			name:               "minor update, pinned",
			extName:            "minor-update-pin",
			extUpdateAvailable: true,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "1.1.0",
			extURL:             "https//github.com/dne/minor-update",
			extIsPinned:        true,
			wantStderr: heredoc.Doc(`
				A new release of minor-update-pin is available: 1.0.0 → 1.1.0
				To upgrade, run: gh extension upgrade minor-update-pin --force
				https//github.com/dne/minor-update
			`),
		},
		{
			name:               "patch update",
			extName:            "patch-update",
			extUpdateAvailable: true,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "1.0.1",
			extURL:             "https//github.com/dne/patch-update",
			wantStderr: heredoc.Doc(`
				A new release of patch-update is available: 1.0.0 → 1.0.1
				To upgrade, run: gh extension upgrade patch-update
				https//github.com/dne/patch-update
			`),
		},
		{
			name:               "patch update, pinned",
			extName:            "patch-update-pin",
			extUpdateAvailable: true,
			extCurrentVersion:  "1.0.0",
			extLatestVersion:   "1.0.1",
			extURL:             "https//github.com/dne/patch-update",
			extIsPinned:        true,
			wantStderr: heredoc.Doc(`
				A new release of patch-update-pin is available: 1.0.0 → 1.0.1
				To upgrade, run: gh extension upgrade patch-update-pin --force
				https//github.com/dne/patch-update
			`),
		},
	}

	for _, tt := range tests {
		ios, _, _, stderr := iostreams.Test()

		em := &extensions.ExtensionManagerMock{
			DispatchFunc: func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (bool, error) {
				// Assume extension executed / dispatched without problems as test is focused on upgrade checking.
				// Sleep for 1 second to allow update checking logic to complete.
				time.Sleep(1 * time.Second)
				return true, nil
			},
		}

		ext := &extensions.ExtensionMock{
			CurrentVersionFunc: func() string {
				return tt.extCurrentVersion
			},
			IsPinnedFunc: func() bool {
				return tt.extIsPinned
			},
			LatestVersionFunc: func() string {
				return tt.extLatestVersion
			},
			NameFunc: func() string {
				return tt.extName
			},
			UpdateAvailableFunc: func() bool {
				return tt.extUpdateAvailable
			},
			URLFunc: func() string {
				return tt.extURL
			},
		}

		checkFunc := func(em extensions.ExtensionManager, ext extensions.Extension) (*update.ReleaseInfo, error) {
			if !tt.extUpdateAvailable {
				return nil, nil
			}

			return &update.ReleaseInfo{
				Version: tt.extLatestVersion,
				URL:     tt.extURL,
			}, nil
		}

		cmd := root.NewCmdExtension(ios, em, ext, checkFunc)

		_, err := cmd.ExecuteC()
		require.NoError(t, err)

		if tt.wantStderr == "" {
			assert.Emptyf(t, stderr.String(), "executing extension command should output nothing to stderr")
		} else {
			assert.Containsf(t, stderr.String(), tt.wantStderr, "executing extension command should output message about upgrade to stderr")
		}
	}
}

func TestNewCmdExtension_UpdateCheckIsNonblocking(t *testing.T) {
	ios, _, _, stderr := iostreams.Test()

	em := &extensions.ExtensionManagerMock{
		DispatchFunc: func(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) (bool, error) {
			// Assume extension executed / dispatched without problems as test is focused on upgrade checking.
			return true, nil
		},
	}

	ext := &extensions.ExtensionMock{
		CurrentVersionFunc: func() string {
			return "1.0.0"
		},
		IsPinnedFunc: func() bool {
			return false
		},
		LatestVersionFunc: func() string {
			return "2.0.0"
		},
		NameFunc: func() string {
			return "major-update"
		},
		UpdateAvailableFunc: func() bool {
			return true
		},
		URLFunc: func() string {
			return "https//github.com/dne/major-update"
		},
	}

	checkFunc := func(em extensions.ExtensionManager, ext extensions.Extension) (*update.ReleaseInfo, error) {
		// This function runs in the background as a goroutine while the extension is dispatched.
		// Testing whether the extension cobra command is non-blocking for extension update check,
		// this function will cause goroutine to sleep for a sufficiently long time before panicking.
		// The idea is that the test will conclude because the dispatch function completes immediately
		// and exits before the check update function completes.
		time.Sleep(30 * time.Second)
		panic("It seems like the extension cobra command might be blocking on update checking when it should not block on long update checks")
	}

	cmd := root.NewCmdExtension(ios, em, ext, checkFunc)

	_, err := cmd.ExecuteC()
	require.NoError(t, err)
	require.Emptyf(t, stderr.String(), "long running, non-blocking extension update check should not have displayed anything on stderr")
}

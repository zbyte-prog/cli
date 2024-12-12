//go:build !windows
package extension

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/stretchr/testify/require"
)

func TestManager_InstallLocal(t *testing.T) {
	dataDir := t.TempDir()
	updateDir := t.TempDir()

	extDir := filepath.Join(t.TempDir(), "gh-local")
	err := os.MkdirAll(extDir, 0755)
	require.NoErrorf(t, err, "failed to create local extension")
	require.NoError(t, stubExtensionUpdate(filepath.Join(updateDir, "gh-local")))

	ios, _, stdout, stderr := iostreams.Test()
	m := newTestManager(dataDir, updateDir, nil, nil, ios)

	err = m.InstallLocal(extDir)
	require.NoError(t, err)
	require.Equal(t, "", stdout.String())
	require.Equal(t, "", stderr.String())

	fm, err := os.Lstat(filepath.Join(dataDir, "extensions", "gh-local"))
	require.NoErrorf(t, err, "data directory missing local extension symlink")
	isSymlink := fm.Mode() & os.ModeSymlink
	require.True(t, isSymlink != 0)
	require.NoDirExistsf(t, filepath.Join(updateDir, "gh-local"), "update directory should be removed")
}

package extension

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/cli/cli/v2/pkg/iostreams"
	"github.com/stretchr/testify/require"
)

func TestManager_InstallLocal_Windows(t *testing.T) {
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

	extDataFile := filepath.Join(dataDir, "extensions", "gh-local")
	fm, err := os.Stat(extDataFile)
	require.NoErrorf(t, err, "data directory missing Windows local extension file")
	isSymlink := fm.Mode() & os.ModeSymlink
	require.True(t, isSymlink == 0)
	extDataFilePath, err := readPathFromFile(extDataFile)
	require.NoErrorf(t, err, "failed to read Windows local extension path file")
	require.Equal(t, extDir, extDataFilePath)
	require.NoDirExistsf(t, filepath.Join(updateDir, "gh-local"), "update directory should be removed")
}

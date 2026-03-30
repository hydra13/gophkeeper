package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewLocalBlob_EmptyBaseDir(t *testing.T) {
	_, err := NewLocalBlob("")
	require.Error(t, err)
	require.Contains(t, err.Error(), "base directory is required")
}

func TestNewLocalBlob_CreatesDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "subdir", "nested")
	blob, err := NewLocalBlob(dir)
	require.NoError(t, err)
	require.NotNil(t, blob)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

func TestSaveRead_Roundtrip(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	data := []byte("hello world")
	require.NoError(t, blob.Save("test.txt", data))

	got, err := blob.Read("test.txt")
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestSave_CreatesSubdirectories(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	data := []byte("nested content")
	require.NoError(t, blob.Save("a/b/c/file.txt", data))

	got, err := blob.Read("a/b/c/file.txt")
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestRead_NonExistentFile(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	_, err = blob.Read("no-such-file.txt")
	require.Error(t, err)
}

func TestDelete_ExistingFile(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	require.NoError(t, blob.Save("to-delete.txt", []byte("data")))
	require.NoError(t, blob.Delete("to-delete.txt"))

	exists, err := blob.Exists("to-delete.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestDelete_NonExistentFile_NoError(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	err = blob.Delete("no-such-file.txt")
	require.NoError(t, err)
}

func TestExists_TrueForExistingFile(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	require.NoError(t, blob.Save("exists.txt", []byte("data")))

	exists, err := blob.Exists("exists.txt")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestExists_FalseForNonExistentFile(t *testing.T) {
	blob, err := NewLocalBlob(t.TempDir())
	require.NoError(t, err)

	exists, err := blob.Exists("no-such-file.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

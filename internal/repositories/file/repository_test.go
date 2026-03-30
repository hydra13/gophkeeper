package file

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- helpers ---

// newTestRepo creates a Repository backed by a temp directory.
func newTestRepo(t *testing.T) *Repository {
	t.Helper()
	repo, err := New(t.TempDir())
	require.NoError(t, err)
	return repo
}

// --- New ---

func TestNew_EmptyBasePath(t *testing.T) {
	t.Parallel()

	repo, err := New("")
	require.Error(t, err)
	require.Nil(t, repo)
	require.Contains(t, err.Error(), "blob base path is required")
}

func TestNew_ValidPath(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "blob")

	repo, err := New(dir)
	require.NoError(t, err)
	require.NotNil(t, repo)
	require.Equal(t, dir, repo.basePath)
}

func TestNew_CreatesDirectory(t *testing.T) {
	t.Parallel()

	dir := filepath.Join(t.TempDir(), "a", "b", "c")

	repo, err := New(dir)
	require.NoError(t, err)
	require.NotNil(t, repo)

	info, err := os.Stat(dir)
	require.NoError(t, err)
	require.True(t, info.IsDir())
}

// --- Save ---

func TestSave_NormalRoundtrip(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	data := []byte("hello world")

	require.NoError(t, repo.Save("file.bin", data))

	got, err := repo.Read("file.bin")
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestSave_EmptyData(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	require.NoError(t, repo.Save("empty.bin", []byte{}))

	got, err := repo.Read("empty.bin")
	require.NoError(t, err)
	require.Equal(t, []byte{}, got)
}

func TestSave_CreatesSubdirectories(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	data := []byte("nested content")

	require.NoError(t, repo.Save("a/b/c/file.txt", data))

	got, err := repo.Read("a/b/c/file.txt")
	require.NoError(t, err)
	require.Equal(t, data, got)
}

func TestSave_OverwritesExistingFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	require.NoError(t, repo.Save("file.txt", []byte("first")))
	require.NoError(t, repo.Save("file.txt", []byte("second")))

	got, err := repo.Read("file.txt")
	require.NoError(t, err)
	require.Equal(t, []byte("second"), got)
}

func TestSave_AtomicNoTmpFileLeft(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	basePath := repo.basePath

	require.NoError(t, repo.Save("file.txt", []byte("data")))

	// No .tmp file should remain in the directory.
	entries, err := os.ReadDir(basePath)
	require.NoError(t, err)
	for _, e := range entries {
		assert.False(t, filepath.Ext(e.Name()) == ".tmp", "unexpected .tmp file left: %s", e.Name())
	}
}

func TestSave_AbsolutePathRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	err := repo.Save("/etc/passwd", []byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestSave_ParentTraversalRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	err := repo.Save("../../../etc/passwd", []byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestSave_ParentTraversalMiddleRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	err := repo.Save("a/../../../etc/passwd", []byte("data"))
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

// --- Read ---

func TestRead_NonExistentFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.Read("no-such-file.bin")
	require.Error(t, err)
}

func TestRead_AbsolutePathRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.Read("/etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestRead_ParentTraversalRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.Read("../../secret")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

// --- Delete ---

func TestDelete_ExistingFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	require.NoError(t, repo.Save("to-delete.bin", []byte("data")))

	require.NoError(t, repo.Delete("to-delete.bin"))

	exists, err := repo.Exists("to-delete.bin")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestDelete_NonExistentFile_NoError(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	err := repo.Delete("no-such-file.bin")
	require.NoError(t, err)
}

func TestDelete_AbsolutePathRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	err := repo.Delete("/etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestDelete_ParentTraversalRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	err := repo.Delete("../../../etc/shadow")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

// --- Exists ---

func TestExists_ExistingFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)
	require.NoError(t, repo.Save("exists.bin", []byte("data")))

	exists, err := repo.Exists("exists.bin")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestExists_NonExistentFile(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	exists, err := repo.Exists("no-such-file.bin")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestExists_AbsolutePathRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.Exists("/etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestExists_ParentTraversalRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.Exists("../../../etc/shadow")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

// --- resolvePath ---

func TestResolvePath_RelativePathOK(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	resolved, err := repo.resolvePath("dir/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repo.basePath, "dir/file.txt"), resolved)
}

func TestResolvePath_CleanedRelativePath(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	// "a/../b" cleans to "b"
	resolved, err := repo.resolvePath("a/../b/file.txt")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repo.basePath, "b/file.txt"), resolved)
}

func TestResolvePath_EmptyPathOK(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	// filepath.Clean("") => "." which is a valid relative path.
	resolved, err := repo.resolvePath("")
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repo.basePath, "."), resolved)
}

func TestResolvePath_AbsolutePathRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.resolvePath("/absolute/path")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestResolvePath_ParentTraversalRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.resolvePath("../etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestResolvePath_DeepParentTraversalRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.resolvePath("a/b/../../../etc/passwd")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

func TestResolvePath_DotDotAloneRejected(t *testing.T) {
	t.Parallel()

	repo := newTestRepo(t)

	_, err := repo.resolvePath("..")
	require.Error(t, err)
	require.Contains(t, err.Error(), "invalid blob path")
}

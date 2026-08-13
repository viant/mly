package triton

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/viant/afs"
	"github.com/viant/afs/storage"
	"github.com/viant/mly/service/config"
)

type countingFS struct {
	afs.Service
	copies atomic.Int32
}

func (c *countingFS) Copy(ctx context.Context, sourceURL, destURL string, options ...storage.Option) error {
	c.copies.Add(1)
	return c.Service.Copy(ctx, sourceURL, destURL, options...)
}

type failingFS struct {
	afs.Service
}

func (f *failingFS) Copy(ctx context.Context, sourceURL, destURL string, options ...storage.Option) error {
	return errors.New("copy failed")
}

func fileURL(path string) string {
	return "file://" + filepath.ToSlash(path)
}

func writeModelTree(t *testing.T, root, modelName string) {
	t.Helper()
	dir := filepath.Join(root, modelName, "1")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, modelName, "config.pbtxt"), []byte("name: \""+modelName+"\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "model.onnx"), []byte("fake"), 0o644))
}

func newTestLocalRepo(t *testing.T, fs afs.Service, localRoot, remoteRoot string) *LocalRepository {
	t.Helper()
	return NewLocalRepository(config.TritonServer{
		LocalModelRepository: localRoot,
		RemoteRepositoryURI:  fileURL(remoteRoot),
		ModelLoadConcurrency: 2,
	}, fs)
}

func TestLocalRepository_EnsureCopiesThenSkips(t *testing.T) {
	remote := t.TempDir()
	local := t.TempDir()
	writeModelTree(t, remote, "modelA")

	counter := &countingFS{Service: afs.New()}
	repo := newTestLocalRepo(t, counter, local, remote)
	ctx := context.Background()

	require.NoError(t, repo.Ensure(ctx, "modelA"))
	require.FileExists(t, filepath.Join(local, "modelA", "config.pbtxt"))
	assert.Equal(t, int32(1), counter.copies.Load())

	require.NoError(t, repo.Ensure(ctx, "modelA"))
	assert.Equal(t, int32(1), counter.copies.Load())
}

func TestLocalRepository_EnsureFailureLeavesNoDest(t *testing.T) {
	remote := t.TempDir()
	local := t.TempDir()
	repo := newTestLocalRepo(t, &failingFS{Service: afs.New()}, local, remote)
	err := repo.Ensure(context.Background(), "modelA")
	require.Error(t, err)
	_, destErr := os.Stat(filepath.Join(local, "modelA"))
	assert.True(t, os.IsNotExist(destErr))
	_, stagingErr := os.Stat(filepath.Join(local, ".modelA.staging"))
	assert.True(t, os.IsNotExist(stagingErr))
}

func TestLocalRepository_RejectsInvalidName(t *testing.T) {
	repo := newTestLocalRepo(t, afs.New(), t.TempDir(), t.TempDir())
	err := repo.Ensure(context.Background(), "../escape")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid Triton model name")
}

func TestService_UnloadModel_RemovesLocalAfterLastUser(t *testing.T) {
	remote := t.TempDir()
	local := t.TempDir()
	writeModelTree(t, remote, "shared")

	mock := &mockTritonClient{readyState: map[string]bool{"shared": false}}
	repo := newTestLocalRepo(t, afs.New(), local, remote)
	svc := NewServiceWithLocalRepository(mock, repo)
	ctx := context.Background()

	svc.RegisterUsage("routerA", "shared")
	svc.RegisterUsage("routerB", "shared")
	require.NoError(t, svc.LoadModel(ctx, "shared"))
	require.FileExists(t, filepath.Join(local, "shared", "config.pbtxt"))

	require.NoError(t, svc.UnloadModel(ctx, "routerA", "shared"))
	require.FileExists(t, filepath.Join(local, "shared", "config.pbtxt"))

	require.NoError(t, svc.UnloadModel(ctx, "routerB", "shared"))
	_, err := os.Stat(filepath.Join(local, "shared"))
	assert.True(t, os.IsNotExist(err))
}

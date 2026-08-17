package triton

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/viant/afs"
	"github.com/viant/afs/option"
	"github.com/viant/mly/service/config"
	"golang.org/x/sync/semaphore"
	"golang.org/x/sync/singleflight"
)

// LocalRepository copies Triton model trees from a remote prefix onto a local
// directory that Triton can later use as --model-repository. Load/unload RPCs
// stay name-only, so copies are valid while Triton still points at the remote URI.
type LocalRepository struct {
	fs        afs.Service
	localRoot string
	remoteURI string
	sema      *semaphore.Weighted
	inflight  singleflight.Group
}

// NewLocalRepository returns a manager when LocalModelRepository is set, else nil.
func NewLocalRepository(server config.TritonServer, fs afs.Service) *LocalRepository {
	if server.LocalModelRepository == "" {
		return nil
	}
	n := int64(server.ModelLoadConcurrency)
	if n <= 0 {
		n = int64(config.DefaultModelLoadConcurrency())
	}
	return &LocalRepository{
		fs:        fs,
		localRoot: server.LocalModelRepository,
		remoteURI: strings.TrimRight(server.RemoteRepositoryURI, "/"),
		sema:      semaphore.NewWeighted(n),
	}
}

func (r *LocalRepository) acquire(ctx context.Context) error {
	return r.sema.Acquire(ctx, 1)
}

func (r *LocalRepository) release() {
	r.sema.Release(1)
}

func modelDirName(modelName string) (string, error) {
	if modelName == "" || modelName != filepath.Base(modelName) || modelName == "." || modelName == ".." {
		return "", fmt.Errorf("invalid Triton model name %q", modelName)
	}
	return modelName, nil
}

func (r *LocalRepository) destPath(modelName string) (string, error) {
	name, err := modelDirName(modelName)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.localRoot, name), nil
}

func (r *LocalRepository) stagingPath(modelName string) (string, error) {
	name, err := modelDirName(modelName)
	if err != nil {
		return "", err
	}
	return filepath.Join(r.localRoot, "."+name+".staging"), nil
}

func (r *LocalRepository) remoteURL(modelName string) (string, error) {
	name, err := modelDirName(modelName)
	if err != nil {
		return "", err
	}
	return r.remoteURI + "/" + name, nil
}

// Ensure copies remoteURI/modelName into localRoot/modelName if that directory
// is not already present. A sibling staging directory is used so Triton never
// sees a half-written tree.
//
// Same-name calls share one in-flight copy. If dest is already there when
// rename runs (another process won the install), that is success: the tree
// is not needed.
func (r *LocalRepository) Ensure(ctx context.Context, modelName string) error {
	_, err, _ := r.inflight.Do(modelName, func() (interface{}, error) {
		return nil, r.ensureOnce(ctx, modelName)
	})
	return err
}

func (r *LocalRepository) ensureOnce(ctx context.Context, modelName string) error {
	dest, err := r.destPath(modelName)
	if err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("stat local Triton model %s: %w", dest, err)
	}

	staging, err := r.stagingPath(modelName)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("remove leftover staging %s: %w", staging, err)
	}
	if err := os.MkdirAll(r.localRoot, 0o755); err != nil {
		return fmt.Errorf("create local Triton model repository %s: %w", r.localRoot, err)
	}

	remote, err := r.remoteURL(modelName)
	if err != nil {
		return err
	}
	options := option.NewSource(&option.NoCache{Source: option.NoCacheBaseURL})
	if err := r.fs.Copy(ctx, remote, staging, options); err != nil {
		_ = os.RemoveAll(staging)
		return fmt.Errorf("copy Triton model %s to %s: %w", remote, staging, err)
	}
	if err := os.Rename(staging, dest); err != nil {
		_ = os.RemoveAll(staging)
		if destInstalled(dest) {
			return nil
		}
		return fmt.Errorf("install Triton model %s: %w", dest, err)
	}
	return nil
}

func destInstalled(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// Remove deletes the local model directory and any leftover staging dir.
func (r *LocalRepository) Remove(modelName string) error {
	dest, err := r.destPath(modelName)
	if err != nil {
		return err
	}
	staging, err := r.stagingPath(modelName)
	if err != nil {
		return err
	}
	if err := os.RemoveAll(staging); err != nil {
		return fmt.Errorf("remove staging %s: %w", staging, err)
	}
	if err := os.RemoveAll(dest); err != nil {
		return fmt.Errorf("remove local Triton model %s: %w", dest, err)
	}
	return nil
}

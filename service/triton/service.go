package triton

import (
	"context"
)

// Service is a container for a client and a representation of model repository management.
type Service struct {
	Client TritonClient

	Unloader   ModelUnloader
	Repository *Repository
	local      *LocalRepository
}

func (s *Service) RegisterUsage(mlyID string, tritonName string) {
	if s.Repository == nil {
		return
	}

	s.Repository.RegisterUsage(mlyModelID(mlyID), tritonModelName(tritonName))
}

func NewService(client TritonClient) *Service {
	return NewServiceWithLocalRepository(client, nil)
}

func NewServiceWithLocalRepository(client TritonClient, local *LocalRepository) *Service {
	return &Service{
		Client:     client,
		Unloader:   client,
		Repository: NewRepository(),
		local:      local,
	}
}

// LoadModel copies the model tree locally when a LocalRepository is configured,
// then issues a name-only RepositoryModelLoad.
func (s *Service) LoadModel(ctx context.Context, modelName string) error {
	if s.local != nil {
		if err := s.local.acquire(ctx); err != nil {
			return err
		}
		defer s.local.release()
		if err := s.local.Ensure(ctx, modelName); err != nil {
			return err
		}
	}

	return s.Client.ModelLoad(ctx, modelName)
}

func (s *Service) UnloadModel(ctx context.Context, mlyID string, tritonName string) error {
	if s.Repository == nil {
		return nil
	}

	if s.Unloader == nil {
		return nil
	}

	shouldUnload := s.Repository.UnregisterUsage(mlyModelID(mlyID), tritonModelName(tritonName))
	if !shouldUnload {
		return nil
	}

	if err := s.Unloader.ModelUnload(ctx, tritonName); err != nil {
		return err
	}

	if s.local != nil {
		return s.local.Remove(tritonName)
	}

	return nil
}

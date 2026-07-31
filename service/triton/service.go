package triton

import (
	"context"
)

// Service is a container for a client and a representation of model repository management.
type Service struct {
	Client TritonClient

	Unloader   ModelUnloader
	Repository *Repository
}

func (s *Service) RegisterUsage(mlyID string, tritonName string) {
	if s.Repository == nil {
		return
	}

	s.Repository.RegisterUsage(mlyModelID(mlyID), tritonModelName(tritonName))
}

func NewService(client TritonClient) *Service {
	return &Service{
		Client:     client,
		Unloader:   client,
		Repository: NewRepository(),
	}
}

func (s *Service) UnloadModel(ctx context.Context, mlyID string, tritonName string) error {
	if s.Repository == nil {
		return nil
	}

	if s.Unloader == nil {
		return nil
	}

	shouldUnload := s.Repository.UnregisterUsage(mlyModelID(mlyID), tritonModelName(tritonName))
	if shouldUnload {
		return s.Unloader.ModelUnload(ctx, tritonName)
	}

	return nil
}

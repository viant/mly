package files

import (
	"context"
	"fmt"

	"github.com/viant/afs"
	"github.com/viant/afs/url"
	"github.com/viant/mly/service/config"
)

// ModifiedSnapshot checks and updates modified times based on the object in URL
func ModifiedSnapshot(ctx context.Context, fs afs.Service, URL string, metadata *config.Modified) (*config.Modified, error) {
	objects, err := fs.List(ctx, URL)
	if err != nil {
		return metadata, fmt.Errorf("failed to list URL:%s; error:%w", URL, err)
	}

	if metadata == nil {
		metadata = new(config.Modified)
	}

	if extURL := url.SchemeExtensionURL(URL); extURL != "" {
		object, err := fs.Object(ctx, extURL)
		if err != nil {
			return nil, err
		}

		metadata.Max = object.ModTime()
		metadata.Min = object.ModTime()
		return metadata, nil
	}

	for i, item := range objects {
		if item.IsDir() && i == 0 {
			continue
		}
		if item.IsDir() {
			metadata, err = ModifiedSnapshot(ctx, fs, item.URL(), metadata)
			if err != nil {
				return metadata, err
			}
			continue
		}
		if metadata.Max.IsZero() {
			metadata.Max = item.ModTime()
		}
		if metadata.Min.IsZero() {
			metadata.Min = item.ModTime()
		}

		if item.ModTime().After(metadata.Max) {
			metadata.Max = item.ModTime()
		}
		if item.ModTime().Before(metadata.Min) {
			metadata.Min = item.ModTime()
		}
	}
	return metadata, nil
}

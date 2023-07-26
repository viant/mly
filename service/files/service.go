package files

import (
	"context"
	"fmt"

	"github.com/viant/afs"
	"github.com/viant/afs/url"
	"github.com/viant/mly/service/config"
)

// ModifiedSnapshot checks and updates modified times based on the object in URL
func ModifiedSnapshot(ctx context.Context, fs afs.Service, URL string, resource *config.Modified) (*config.Modified, error) {
	objects, err := fs.List(ctx, URL)
	if err != nil {
		return resource, fmt.Errorf("failed to list URL:%s; error:%w", URL, err)
	}

	if extURL := url.SchemeExtensionURL(URL); extURL != "" {
		object, err := fs.Object(ctx, extURL)
		if err != nil {
			return nil, err
		}

		resource.Max = object.ModTime()
		resource.Min = object.ModTime()
		return resource, nil
	}

	for i, item := range objects {
		if item.IsDir() && i == 0 {
			continue
		}
		if item.IsDir() {
			resource, err = ModifiedSnapshot(ctx, fs, item.URL(), resource)
			if err != nil {
				return resource, err
			}
			continue
		}
		if resource.Max.IsZero() {
			resource.Max = item.ModTime()
		}
		if resource.Min.IsZero() {
			resource.Min = item.ModTime()
		}

		if item.ModTime().After(resource.Max) {
			resource.Max = item.ModTime()
		}
		if item.ModTime().Before(resource.Min) {
			resource.Min = item.ModTime()
		}
	}
	return resource, nil
}

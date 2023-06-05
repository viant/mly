package main

import (
	"os"

	"github.com/viant/mly/example/client"
	slfmodel "github.com/viant/mly/example/transformer/slf/model"
	"github.com/viant/mly/shared/common"
	"github.com/viant/mly/shared/common/storable"
)

func main() {
	storableSrv := storable.Singleton()

	// in actual client code, any type information should be available within the context of the caller, so using a dynamic
	// type instantiation service is not needed
	storableSrv.Register(slfmodel.Namespace, func() common.Storable {
		return new(slfmodel.Segmented)
	})

	storableSrv.Register("slft_batch", func() common.Storable {
		sa := make([]*slfmodel.Segmented, 0)
		segmenteds := slfmodel.Segmenteds(sa)
		return &segmenteds
	})

	client.Run(os.Args[1:])
}

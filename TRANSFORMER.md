# Transformer 

By default, the model signature output name alongside the model prediction gets used to produce cachable output.
This can be customized by implementing a custom transformer.

A custom transformer has to use the following function signature:

```go
type Transformer func(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error)
```

Then to register the transformer:

```go
import "github.com/viant/mly/service/domain/transformer"

func init() {
  transformer.Register("myTransformer", aTransformer)
}
```

Optionally you can implement a `storable` provider.

```go
import "github.com/viant/mly/service/domain/transformer"

func init() {
  transformer.Register("myType", func() interface{} {
      return &MyOutputType{}
  })
}
```

Where `MyOutputType` could implement the following interfaces to avoid reflection:
- [Storable](shared/common/storable.go) (Aerospike storage)
- [Bintly](https://github.com/viant/bintly) (in-memory serialization)
- [Gojay JSON](https://github.com/francoispqt/gojay/) (HTTP response)
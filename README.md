# Mly - machine learning executor on steroids for golang

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/mly)](https://goreportcard.com/report/github.com/viant/mly)
[![GoDoc](https://godoc.org/github.com/viant/mly?status.svg)](https://godoc.org/github.com/viant/mly)

This library is compatible with Go 1.16+

- [Motivation](#motivation)
- [Introduction](#introduction)
- [Configuration](#configuration)  
- [Usage](#usage)
- [Contribution](#contributing-to-mly)
- [License](#license)

## Motivation

The goal of this library to provide a generic TensorFlow wrapper web service which can speed up end to end execution by leveraging a caching system. 
Caching has to be seamless, meaning the client, behind the scenes, should take care of any dictionary-based key generation and model changes automatically. 

In practice this library can provide substantial (100x) E2E execution improvement from the client side perspective, with the input space of billions of distinct keys. 

Each model provides both TensorFlow and cache-level performance metrics via HTTP REST API.

## Introduction

This project provides libraries for both the client and the web service.

The web service supports multiple TensorFlow model integrations on the URI level, with `GET`, `POST`  method support with HTTP 2.0 in full duplex mode as provided by Golang libraries.

The service automatically detects and reloads any model changes; it will poll the source files for modifications once a minute.
Technically, any HTTP client can work with the service, but to get the seamless caching benefit, it's recommended to use provided client.

## Quickstart

To start a HTTP service with 2 models, from the repository root:

1. Create a `config.yaml` file:

```yaml
Endpoint:
  Port: 8086
Models:
  - ID: ml0
    URL: gs://modelBucket/Ml0ModelFolder
    OutputType: float32
  - ID: mlx
    URL: gs://modelBucket/MlXModelFolder
    OutputType: float32
```

2. Start the example server (in the background) with `go run ./example/server -c config.yaml`.
3. Then invoke a prediction with `curl 'http://localhost:8086/v1/api/model/ml0/eval?modelInput1=val1&modelInputX=valueX'`.

## Caching

The main performance benefit comes from trading compute with space.

In order to leverage caching, the model has to use categorical features with a fixed vocabulary, with the input space providing a reasonable cache hit rate.

Currently, only `StringLookup` and `IntegerLookup` layers are supported for caching.
In terms of practical limits, only models with categorical features can be cached; even numeric values can theoretically be cached, if a given level of input precision loss is acceptable.

By default, the client will configure itself using the web service cache settings.  
This enables the `mly` client to handle key generation without additional configuration or code.

The library supports 3 types of caching:
- in-(process) memory
- external Aerospike cache
- hybrid

When the in-memory cache is used, both the client and web service manage their cache independently; meaning if data is missing in the client cache, the client will send a request to server endpoint.

When an external cache is used, the client will first check the external cache that is shared with web service; if data is found, it's copied to local in-memory cache. 

The in-memory cache uses [scache](https://github.com/viant/scache)'s most-recently-used implementation.

**Example of `config.yaml` with in-memory cache**

```yaml
Endpoint:
  Port: 8086

Models:
  - ID: mlX
    URL: s3://myBucket/myModelX
    OutputType: float32
    Datastore: mem

Datastores:
  - ID: mem
    Cache:
      SizeMB: 100
```

To deal with larger key spaces, an external cache can be further configured using a tiered caching strategy.
Any cached value will propagate upwards once found.

For example, we can have a 2 tier caching strategy, where we will call the tiers L1 and L2.
In this scenario, the L2 cache can be a very large SSD-backed Aerospike instance and L1 cache could be a smaller memory-based instance. 

In this case, when we look for a cached value, first the in-memory cache is checked, followed by L1, then L2. 
Once the value is found, the value is copied to L2 then from L2 to L1 and L1 to local memory, depending on where it's found.


**Example of `config.yaml` with both an in-memory and an Aerospike cache**

```yaml
Endpoint:
  Port: 8080

Models:
  - ID: mlv
    URL: s3://myBucket/myModelX
    OutputType: float32
    Datastore: mlxCache

Datastores:
  - ID: mlxCache
    Cache:
      SizeMB: 1024
    Connection: local
    Namespace: udb
    Dataset: mlvX

Connections:
  - ID: local
    Hostnames: 127.0.0.1
```

### Dictionary hash code

In caching mode, in order to manage cache and client/server consistency every time a model/dictionary gets re/loaded, `mly` computes a dictionary hash code.
This hash code gets stored in the cache along with model prediction and is passed to the client in every response. 
Once a client detects a change in dictionary hash code, it automatically initiates a dictionary reload and invalidates cache entries.

## Configuration

### Server

The server accepts configuration with the following options:

* `Models` : list of models
  - `ID`: required model ID
  - `URL`: required model location  
    * to use S3, set environment variable `AWS_SDK_LOAD_CONFIG=true`
    * to use GCS, set environment variable `GOOGLE_APPLICATION_CREDENTIALS=true`
  - `Tags`: optional model tags (default "serve")
  - `OutputType`: required output type (`int64`, `float32`, etc..)
  - `UseDict`: optional flag to use dictionary/caching (`true` by default)
  - `KeyFields`: optional list of fields used to generate caching key (by default, all model inputs sorted alphabetically)
  - `DataStore`: optional name of datastore cache
  - `Transformer`: optional name of model output transformer
    
* `Connection` : optional list of external Aerospike connections
  - `ID`: required connection ID
  - `Hostnames`: required Aerospike hostnames

* `Datastores` : list of datastore caches
  - `ID`: required datastore ID (to be matched with `Model.DataStore`)
  - `Connection`: optional connection ID
  - `Namespace`: optional Aerospike namespace
  - `Dataset`: optional Aerospike dataset 
  - `Storable`: name of registered `storable` provider
  - `Cache`: optional in-memory cache setting
      * `SizeMB`: optional cache size in MB

### Client
 
`mly` client does not come with external config file.

To create a client, use the following snippet:

```go
mly := client.New("$modelID", []*client.Host{client.NewHost("mlServiceHost", mlServicePort)}, options ...)
```

Where optional `options` can be of the following:
  * `NewCacheSize(sizeOption)`
  * `NewCacheScope(CacheScopeLocal|CacheScopeL1|CacheScopeL2)`
  * `NewGmetric()` - custom instance of `gmetric` service

## Usage

### Server

To code a server executable you can use the following code:

```go
package main

import (
	"github.com/viant/mly/service/endpoint"
	"os"
)

const (
	Version = "1.0"
)

func main() {
	endpoint.RunApp(Version, os.Args)
}
```

### Client

```go
package main

import (
  "context"
  "fmt"
  "github.com/viant/mly/shared/client"
  "log"
)

type Prediction struct {
  Output float32
}

func main() {
  mly, err := client.New("$modelID", []*client.Host{client.NewHost("mlyEndpointHost", 8080)})
  if err != nil {
    log.Fatal(err)
  }
  response := &client.Response{Data: &Prediction{}}
  msg := mly.NewMessage()
  msg.StringKey("input1", "val1")
  //....
  msg.IntKey("inputN", 1)

  err = mly.Run(context.TODO(), msg, response)
  if err != nil {
    log.Fatal(err)
  }
  fmt.Printf("response: %+v\n", response)
}
```

### Output post-processing

By default, the model signature output name alongside the model prediction gets used to produce cachable output.
This process can be customized for specific needs.

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

## Endpoints

###  `/v1/api/config` 

Shows the loaded and processed configuration.

### `/v1/api/health` 

Shows if any models are failing to reload.
Payload is a JSON object whose keys are each model ID as specified in the `config.yaml`, with values a number, where 0 indicates a failure to reload and 1 indicates that the last attempted reload was successful.

#### Example

For a `config.yaml` like:

```yaml
Endpoint:
  Port: 8086
Models:
  - ID: ml0
    URL: gs://modelBucket/Ml0ModelFolder
    OutputType: float32
  - ID: mlx
    URL: gs://modelBucket/MlXModelFolder
    OutputType: float32
```

The endpoint will provide a response like:

```
{
  "ml0": 1,
  "mlx": 1
}
```

### `/v1/api/metric/operations` 

All metrics registered in the web service.
These are provided via [`gmetric`](https://github.com/viant/gmetric).

In all these, `%s` is `Model[].ID` (i.e. from `config.yaml`)

- `/v1/api/metric/operation/%sPerf` - Records metrics related to model handlers (compare with the related `Eval` metrics to calculate overhead).
- `/v1/api/metric/operation/%sEval` - Records metrics related to the TensorFlow operations.

### `/v1/api/model`

Model operations.

In all these, `%s` is `Model[].ID` (i.e. from `config.yaml`)

- `/v1/api/model/%s/eval` - runs `GET` / `POST` model prediction.
- `/v1/api/model/%s/meta/config` - provides configuration for client related to model
- `/v1/api/model/%s/meta/dictionary` - provides current dictionary

## Client Metrics (`gmetric`)

These are provided via [`gmetric`](https://github.com/viant/gmetric).

- `%sperformance` where `%s` is the datastore ID, i.e. `DataStores[].ID` from `config.yaml`.
- `%sClient` where `%s` is the model ID, i.e. `Models[].ID` from `config.yaml`.

## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

## Contributing to `mly`

`mly` is an open source project and contributors are welcome!

See [TODO](TODO.md) list.

## Credits and Acknowledgements

**Library Author:** Adrian Witas


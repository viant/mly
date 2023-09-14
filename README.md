# mly - Latency focused deep-learning model runner client and server

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/mly)](https://goreportcard.com/report/github.com/viant/mly)
[![GoDoc](https://godoc.org/github.com/viant/mly?status.svg)](https://godoc.org/github.com/viant/mly)

This library is compatible with Go 1.17+

- [Motivation](#motivation)
- [Introduction](#introduction)
- [Configuration](#configuration)  
- [Usage](#usage)
- [Contribution](#contributing-to-mly)
- [License](#license)

# Introduction

The goal of this library to provide a deep-learning model prediction HTTP service which can speed up end to end execution by leveraging a caching system. 
Currently the only deep-learning library supported is TensorFlow.

The client cares of any dictionary-based key generation and model changes automatically. 

In practice this library can provide substantial (100x) E2E execution improvement from the client side perspective, with the input space of billions of distinct keys. 

Each model provides both TensorFlow and cache-level performance metrics via HTTP REST API.

This project provides libraries for both the client and the web service.

The web service supports multiple TensorFlow model integrations on the URI level, with `GET`, `POST`  method support with HTTP 2.0 in full duplex mode as provided by Golang libraries.

The service automatically detects and reloads any model changes; it will poll the source files for modifications once a minute.
Technically, any HTTP client can work with the service, but the provided client provides extra caching support.

# Quickstart

To start a HTTP service with a model, from the repository root:

1. Create a `config.yaml` file:

```yaml
Endpoint:
  Port: 8086
Models:
  - ID: ml0
    URL: /path/to/model/ml0
```

The `URL` is loaded using [afs](https://github.com/viant/afs).

2. Start the example server in the background with `go run ./example/server -c config.yaml &`.
3. Then invoke a prediction with `curl 'http://localhost:8086/v1/api/model/ml0/eval?modelInput1=val1&modelInputX=valueX'`.
4. Bring the server to the foreground by using the command `fg` .
5. Use Ctrl+C to terminate the server.

# Caching

The main performance benefit comes from trading compute with space.

In order to leverage caching, the model has to use categorical features with a fixed vocabulary, with the input space providing a reasonable cache hit rate.

As of `v0.8.0`, numeric features are supported. 
The cache space is reduced by limiting decimal precision.

Until `v0.8.0`, only `StringLookup` and `IntegerLookup` layers are supported for caching.
In terms of practical limits, only models with categorical features can be cached; even numeric values can theoretically be cached, if a given level of input precision loss is acceptable.

By default, the client will configure itself using the web service cache settings.  
This enables the `mly` client to handle key generation without additional configuration or code.

The library supports 3 types of caching:
- in-(process) memory
- external Aerospike cache
- hybrid

The in-memory cache uses [scache](https://github.com/viant/scache)'s most-recently-used implementation.

When an external cache is used, the client will first check the external cache that is shared with web service; if data is found, it's copied to local in-memory cache. 

To deal with larger key spaces, an external cache can be further configured using a tiered caching strategy.
Any cached value will propagate upwards once found.

For example, we can have a 2 tier caching strategy, where we will call the tiers L1 and L2.
In this scenario, the L2 cache can be a very large SSD-backed Aerospike instance and L1 cache could be a smaller memory-based instance. 

In this case, when we look for a cached value, first the in-memory cache is checked, followed by L1, then L2. 
Then with a cache miss, the value is calculated then copied to L2 - then from L2 to L1 and L1 to local memory.

**Example of `config.yaml` with both an in-memory and an Aerospike cache**

```yaml
Endpoint:
  Port: 8080

Models:
  - ID: mlx
    URL: /path/to/myModelX
    Datastore: mlxCache

Datastores:
  - ID: mlxCache
    Connection: local
    Namespace: udb
    Dataset: mlX

Connections:
  - ID: local
    Hostnames: 127.0.0.1
```

## Dictionary hash code

In caching mode, in order to manage cache and client/server consistency every time a model/dictionary gets re/loaded, `mly` computes a dictionary hash code.
This hash code gets stored in the cache along with model prediction and is passed to the client in every response. 
Once a client detects a change in dictionary hash code, it automatically initiates a dictionary reload and invalidates cache entries.

# Configuration

## Server

See [`service/endpoint/config.go`](service/endpoint/config.go).
The server accepts configuration with the following options:

* `Models` : list of models - see [`service/config/model.go`](service/config/model.go) for all options.
  - `ID`: `string` - required - model ID, used to generate the URLs.
  - `Debug`: `bool` - optional - enables further output and debugging.
  - `URL`: `string` - required - model location source.
    * to use S3, set environment variable `AWS_SDK_LOAD_CONFIG=true`
    * to use GCS, set environment variable `GOOGLE_APPLICATION_CREDENTIALS=true`
  - `DataStore`: `string` - optional - name of Datastore to cache, should match `Datastores[].ID`.
  - `Transformer`: `string` - optional - name of model output transformer. See [#Transformer](#Transformer).
  - `Batch`: optional - enables or overrides server-side batching configuration. See [`service/tfmodel/batcher/config/config.go`](service/tfmodel/batcher/config/config.go).
  - `Test`: optional - enables a client request to send to self on start up.
    * `Test`: `bool` - if `true`, a client will generate a non-batch request with random values based on the model input signature.
    * `Single`: `map[string]interface{}` - if present, will use the provided values for certain input keys, otherwise randomly generated based on model input signature.
    * `SingleBatch`: `bool` - if `true`, a client will generate a batch request with random values based on the model input signature; if `Single` is set, values will be used for provided keys.
    * `Batch`: `map[string][]interface{}` - if present, will be used to generate a batch of requests for the self-test.
  - `Inputs`: optional - used to further provide or define inputs, a list of `shared.Field`.
    * `Name`: `string` - required - input name, only required if an entry is provided.
    * `Index`: `int` - optional - used to maintain cache key ordering.
    * `Auxiliary`: `bool` - optional - the input is permitted to be provided in an evaluation request.
    * `Wildcard`: `bool` - conditionally required - if enabled this input will not have a vocabulary for lookup; if `UseDict` is true, the service will refuse to start if it cannot guess the vocabulary extraction Operation. 
    * `Precision`: `int` - conditionally required - if the input is a float type and dictionary is enabled, this can be used to round the value to a lower precision which can improve cache hit rates; if `UseDict` is true, the service will refuse to start if it encounters a float input without a `Precision`.
  - `KeyFields`: `[]string` - optional - list of fields used to generate caching key (by default, all model inputs, sorted alphabetically). Can be used to order and add valid inputs that can be used as a cache key but not used as prediction input.
  - `Auxiliary`: `[]string` - deprecated, optional - list of additional fields that are acceptable for eval server call. Deprecated, use `Field.Auxiliary`.
  - `Outputs`: `[]shared.Field` - deprecated, optional - model outputs are automatically pulled from the model.
    
* `Connection`: optional - list of external Aerospike connections.
  - `ID`: `string` - required - connection ID
  - `Hostnames`: `string` - required - Aerospike hostnames

* `Datastores` : list of datastore caches
  - `ID`: `string` - required - datastore ID (to be matched with `Models[].DataStores[].ID`)
  - `Connection`: `string` - optional - connection ID
  - `Namespace`: `string` - optional - Aerospike namespace
  - `Dataset`: `string` - optional - Aerospike dataset 
  - `Storable`: `string` - optional - name of registered `storable` provider
  - `Cache`: optional - in-memory cache setting
    * `SizeMB`: `int` - optional - cache size in MB

* `Endpoint`: some special administrative options
  - `Port`: `int` - optional - used in `addr` for `http.Server`, default `8080`.
  - `ReadTimeoutMs`, `WriteTimeoutMs`: `int` - optional - additional settings for `http.Server`, default `5000` for both.
  - `MaxHeaderBytes`: `int` - optional - additional settings for `http.Server`, default `8192` (`8 * 1024`).
  - `WriteTimeout`: `int` - optional - maximum request timeout.
  - `PoolMaxSize`, `BufferSize`: `int` - optional - controls implementation of `net/http/httputil`, default `512` and `131072` (`128 * 1024`), respectively.
  - `MaxEvaluatorConcurrency`: `int` - optional - controls semaphore that prevents too many CGo goroutines from spawning, default `5000`.

* `EnableMemProf`: `bool` - optional - enables endpoint for memory profiling.
* `EnableCPUProf`: `bool` - optional - enables endpoint for cpu profiling.
* `AllowedSubnet`: `bool` - optional - restricts administrative endpoints to IP string prefixes.
  - Restricts the system configuration, memory profile, CPU profile, and health endpoints.

## Client
 
`mly` client does not come with an external config file.

To create a client, use the following snippet:

```go
mly := client.New("$modelID", []*client.Host{client.NewHost("mlServiceHost", mlServicePort)}, options ...)
```

Where optional `options` can be of, but not limited to, the following:
  * `NewCacheSize(sizeOption)`
  * `NewCacheScope(CacheScopeLocal|CacheScopeL1|CacheScopeL2)`
  * `NewGmetric()` - custom instance of `gmetric` service

See [`shared/client/option.go`](shared/client/option.go) for more options. 

# Usage

## Server

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

## Client

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

# Transformer 

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

# Server Endpoints

##  `/v1/api/config` 

Shows the loaded and processed configuration.

## `/v1/api/health` 

Shows if any models are failing to reload.
Payload is a JSON object whose keys are each model ID as specified in the `config.yaml`, with values a number, where 0 indicates a failure to reload and 1 indicates that the last attempted reload was successful.

### Example

For a `config.yaml` like:

```yaml
Endpoint:
  Port: 8086
Models:
  - ID: ml0
    URL: gs://modelBucket/Ml0ModelFolder
  - ID: mlx
    URL: gs://modelBucket/MlXModelFolder
```

The `/v1/api/health` endpoint will provide a response like:

```
{
  "ml0": 1,
  "mlx": 1
}
```

## `/v1/api/metric/operations` 

All metrics registered in the web service.
These are provided via [`gmetric`](https://github.com/viant/gmetric).

In all these, `%s` is `Model[].ID` (i.e. from `config.yaml`)

- `/v1/api/metric/operation/%sPerf` - Records metrics related to model handlers (compare with the related `Eval` metrics to calculate overhead).
- `/v1/api/metric/operation/%sEval` - Records metrics related to the TensorFlow operations.
- `/v1/api/metric/operation/%sDictMeta` - Records metrics to client dictionary fetch.
- `/v1/api/metric/operation/%sCfgMeta` - Records metrics to client configuration fetch.
- `/v1/api/metric/operation/%sMetaHandler` - Records server-side metrics to client set up.

## `/v1/api/debug`

Requires `EnableMemProf` and / or `EnableCPUProf` to be enabled. 
See [`service/endpoint/prof.go`](service/endpoint/prof.go) for details - otherwise, refer to `pprof` documentation.

## `/v1/api/model`

Model operations.

In all these, `%s` is `Model[].ID` (i.e. from `config.yaml`)

- `/v1/api/model/%s/eval` - runs `GET` / `POST` model prediction.
- `/v1/api/model/%s/meta/config` - provides configuration for client related to model
- `/v1/api/model/%s/meta/dictionary` - provides current dictionary

# Client Metrics (`gmetric`)

These are provided via [`gmetric`](https://github.com/viant/gmetric).

- `%s` where `%s` is the datastore ID, i.e. `DataStores[].ID` from `config.yaml`.
- `%sClient` where `%s` is the model ID, i.e. `Models[].ID` from `config.yaml`.

# License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

# Contributing to `mly`

`mly` is an open source project and contributors are welcome!

# Credits and Acknowledgements

**Initial Author:** Adrian Witas
**Current Author:** David Choi

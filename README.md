# mly - Latency focused deep-learning model runner client and server

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/mly)](https://goreportcard.com/report/github.com/viant/mly)
[![GoDoc](https://godoc.org/github.com/viant/mly?status.svg)](https://godoc.org/github.com/viant/mly)

This library is compatible with Go 1.22+

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

# Quick Start

To start a HTTP service with a model, from the repository root:

### 1 Have a TensorFlow model

If you don't have a TensorFlow model, this is most likely an inappropriate use case for this application.
Alternatively, you can use the provided example models in `example/model`.

### 2 Install TensorFlow

Install TensorFlow following the [Golang Install Guide](https://github.com/tensorflow/build/tree/master/golang_install_guide).
Note there are some caveats with `libtensorflow` installations outside `/usr/local/lib`.

### 3 Configuration

Create a `config.yaml` file:

```yaml
Endpoint:
  Port: 8086
Models:
  - ID: sli
    URL: /path/to/repo/example/model/string_lookups_int_model
```

Note: The `URL` is loaded using [afs](https://github.com/viant/afs).

### 4 Run the server

Start the example server in the background with:

```bash
go run ./example/server/mly -c config.yaml &
```

If you did not [use the Linker](https://www.tensorflow.org/install/lang_c#linker), you need to specify the appropriate Cgo flags:

Note: Replace `$PATH_TO_LIBTENSORFLOW` with your TensorFlow installation.

```bash
# for go build, it is recommended to use LD_LIBRARY_PATH (or DYLD_LIBRARY_PATH on macOS) instead of -Wl,-rpath,$PATH_TO_LIBTENSORFLOW/lib
CGO_CFLAGS="-I$PATH_TO_LIBTENSORFLOW/include" \
CGO_LDFLAGS="-L$PATH_TO_LIBTENSORFLOW/lib  -Wl,-rpath,$PATH_TO_LIBTENSORFLOW/lib" \
go run ./example/server/mly -c config.yaml &
```

### 5 Invoke a prediction

Invoke a prediction with 

```bash
curl 'http://localhost:8086/v1/api/model/sli/eval?sa=v2&sl=v3&aux=v4&kf=v1'
```

### 6 Terminate the server

Bring the server to the foreground by using the command `fg` .
Use Ctrl+C to terminate the server.

# Caching

The main performance benefit comes from trading compute with space.

In order to leverage caching, the model has to use categorical features with a fixed vocabulary, with the input space providing a reasonable cache hit rate.

Categorical features can be cached, and out-of-dictionary values will be cached using the `UNK` token.
Numerical features can be cached limiting decimal precision, otherwise it is not recommended to leverage the cache for models with numerical features.

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

[See `CONFIG.md`](CONFIG.md).

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

By default, the model signature outputs the layer names alongside the model prediction to produce cachable output.

See [`TRANSFORMER.md`](TRANSFORMER.md) for more details.

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

TODO - Add more metrics added from server-side batching.

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

# Versioning Notes

## API Surface

In semantic versioning, it is required to define what the API surface covers, which will include:

* Public Go modules, structs, functions, etc.
* HTTP endpoints, including metric endpoints.
* Command line flags, environment variables, [configuration file](service/endpoint/config.go) and formats.
* [Data log entity schema](service/stream/service.go).
* Edge case behavior changes, by default, will be a patch version bump.
* Bug fixes will be a patch version bump.

### Other notes

Cache key generation logic is permanently tied to the client version.
Changes to the cache key generation logic will result in a new client module version, and semantic versioning will follow accordingly.

## Noted versions

- `v0.14.4` - migrated TensorFlow Go from `tensorflow/tensorflow` to `wamuir/graft`
- `v0.14.2` - go 1.17 to go 1.22
- `v0.8.0` - numeric features are supported

Until `v0.8.0`, only `StringLookup` and `IntegerLookup` layers are supported for caching.

# Credits and Acknowledgements

**Initial Author:** Adrian Witas
**Current Author:** David Choi

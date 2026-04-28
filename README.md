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

See [WORKFLOW.md](WORKFLOW.md) for Mermaid diagrams explaining the Client and more complex caching workflows.

## Dictionary hash code

In caching mode, in order to manage cache and client/server consistency every time a model/dictionary gets re/loaded, `mly` computes a dictionary hash code.
This hash code gets stored in the cache along with model prediction and is passed to the client in every response. 
Once a client detects a change in dictionary hash code, it automatically initiates a dictionary reload and invalidates cache entries.

Note: The dictionary hash code is stored under a special key in Aerospike defined in `shared/common.HashBin`. To prevent conflicts, do not use that same key name for storing your own model predictions.

# Configuration

[See `CONFIG.md`](CONFIG.md), which also includes Client-side configurations.

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

As noted before, see [WORKFLOW.md](WORKFLOW.md) for Mermaid diagrams explaining the Client and more complex caching workflows.

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
- `/v1/api/metric/operation/%sHTTPHandler` - Records per-request HTTP handler metrics. Includes `responseMarshalError` (response struct could not be marshaled; server returned 500) and `responseCommittedError` (body write failed after status was committed; client sees `200 OK` with a body shorter than `Content-Length`).

## `/v1/api/debug`

Requires `EnableMemProf` and / or `EnableCPUProf` to be enabled. 
See [`service/endpoint/prof.go`](service/endpoint/prof.go) for details - otherwise, refer to `pprof` documentation.

In the following sections, `%s` is `Model[].ID` (i.e. from `config.yaml`).

## `/v1/api/model/%s/eval`

Runs a model prediction. This is the primary data-plane endpoint and the only one with a structured request and response shape; the others are administrative or metadata.

### Methods

- `GET` - input values supplied as URL query parameters. Single-prediction mode only.
- `POST` - JSON body containing input values plus optional `batch_size` and `cache_key`. Supports both single-prediction and batch mode.

### Request body (POST)

A JSON object whose keys are model input names (as defined by the model's signature). Values are either scalars (single mode) or arrays of length `batch_size` or `1` (batch mode). Two reserved optional keys:

- `batch_size` (integer, optional) - if present and `> 0`, switches the request to batch mode. Other input values must then be arrays.
- `cache_key` (string in single mode, string array of length `batch_size` in batch mode, optional) - explicit cache key(s) to use instead of letting the server derive one from the input values.

A minimal single-mode request:

```json
{"input1": "value1", "input2": 42}
```

A batch-mode request with two predictions and explicit cache keys:

```json
{
  "batch_size": 2,
  "cache_key": ["k1", "k2"],
  "input1": ["value1", "value2"],
  "input2": [42, 43]
}
```

### Successful response

`200 OK` with `Content-Type: application/json` and an explicit `Content-Length`. The body is a JSON object:

```json
{"status": "ok", "dictHash": 12345, "data": {...}, "serviceTimeMcs": 1100}
```

- `status` - always `"ok"` on success.
- `dictHash` - hash of the dictionary the prediction was made against. Clients use this to detect dictionary changes and trigger a reload (see [Dictionary hash code](#dictionary-hash-code)).
- `data` - the model output, shape determined by the model and any registered transformer.
- `serviceTimeMcs` - server-side time spent on this request in microseconds.

A short read against the declared `Content-Length` indicates a transport failure (peer closed mid-response, broken pipe, etc.), not a successful empty response. Clients should surface short reads as errors rather than treating them as empty bodies.

### Error response

Errors return a non-2xx HTTP status code with the same `Content-Type: application/json` and explicit `Content-Length` as the success response. The body is the same `Response` JSON object with `status` set to `"error"`, the `error` field populated, and `data` omitted:

```json
{"status": "error", "error": "<message>", "serviceTimeMcs": 1100}
```

| status | cause |
| ------ | ----- |
| `400 Bad Request` | malformed query string, malformed JSON body, type mismatch on an input value, or any client-side input error |
| `413 Request Entity Too Large` | POST body exceeds the server's request buffer |
| `429 Too Many Requests` | server is overloaded (evaluator queue rejected the request) |
| `500 Internal Server Error` | prediction failure, server-side encoding failure, or any other server-side error |

Clients can therefore parse the response body the same way regardless of HTTP status — the only differences are the status code and which fields are populated. As a fallback for the rare case where the server cannot encode an error response, a plain-text body may be returned with the same status code.

### Example

```bash
# GET, single prediction
curl 'http://localhost:8086/v1/api/model/ml0/eval?input1=value1&input2=42'

# POST, batch prediction
curl -X POST 'http://localhost:8086/v1/api/model/ml0/eval' \
  -H 'Content-Type: application/json' \
  -d '{"batch_size":2,"input1":["a","b"],"input2":[1,2]}'
```

## `/v1/api/model/%s/meta/config`

Returns the client configuration derived from the model (cache settings, input/output schema, etc.). Used by `mly` clients to bootstrap.

## `/v1/api/model/%s/meta/dictionary`

Returns the current dictionary (categorical input vocabularies + dictionary hash). Used by `mly` clients to populate the local cache and detect dictionary changes.

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

- `v0.20.0` - error responses from `/v1/api/model/%s/eval` are now JSON-encoded `Response` objects (same `Content-Type` / `Content-Length` contract as success responses) instead of plain text; HTTP status codes are unchanged. The client populates `response.Error` from the parsed body, so consumers can rely on either the `err` return value or `response.Error` as the error signal.
- `v0.14.1` last support for go 1.17
- `v0.8.0` - numeric features are supported. 

Until `v0.8.0`, only `StringLookup` and `IntegerLookup` layers are supported for caching.

# Credits and Acknowledgements

**Initial Author:** Adrian Witas
**Current Author:** David Choi

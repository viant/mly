# Mly - machine learning executor on steroids for golang

[![GoReportCard](https://goreportcard.com/badge/github.com/viant/mly)](https://goreportcard.com/report/github.com/viant/mly)
[![GoDoc](https://godoc.org/github.com/viant/mly?status.svg)](https://godoc.org/github.com/viant/mly)

This library is compatible with Go 1.16+

Please refer to [`CHANGELOG.md`](CHANGELOG.md) if you encounter breaking changes.

- [Motivation](#motivation)
- [Introduction](#introduction)
- [Configuration](#configuration)  
- [Usage](#usage)
- [Contribution](#contributing-to-mly)
- [License](#license)

## Motivation

The goal of this library to provide a generic TensorFlow wrapper web service which can speed up end to end execution by leveraging a caching system. 
Caching has to be seamless, meaning the client, behind the scenes, should take care of any dictionary-based key generation and model changes automatically. 

In order to get te benefit of cache, the model dictionary has to use enumerated layers, with the keyspace providing a reasonable cache hit rate.
In practice this library can provide substantial (100x) e2e execution improvement from the client side perspective, with the keyspace ranging from millions of billions distinct keys. 

Performance transparency: each model provides both TensorFlow and cache-level performance metrics exposed via REST API.

## Introduction

This project provides libraries for both the client and the ML web service.
The service supports multiple TensorFlow model integrations on the URI level, with GET/POST method support, as well as PUT with HTTP 2.0 in full duplex mode. 
The service automatically detects and reloads any model changes.
Technically, any HTTP client can work with the service, but to get the seamless caching benefit, it's recommended to use provided client.

To start HTTP service with 2 models, create a service `config.yaml` file, so you can run GET/POST prediction using the following URL: 

```
http://mlyHost:8086/v1/api/model/$MODEL_ID/eval?modelInput1=val1&modelInputX=valueX
```

`config.yaml`
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


#### Caching

The main performance benefit comes from trading CPU with I/O.
In order to use the cache, a client has to generate the caching key.
The native `mly` client handles key generation and passing automatically.

Depending on keyspace size, the library supports 3 levels of caching.

- in-(process) memory
- external Aerospike cache
- hybrid

By default, the client inherits the web service cache settings.

* When the in-memory cache is used, both the client and web service manage their cache independently; meaning if data is missing in 
the client cache, the client will send a request to server endpoint.

* When an external cache is used, the client will first check the external cache that is shared with web service; if data is found, 
it's copied to local in-memory cache. 

The in-memory cache uses [scache](https://github.com/viant/scache)'s most-recently-used implementation.

**Example of service with in-memory cache**

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

To deal with large keyspaces, the external cache can be further configured using a L1 and L2 cache. 
In this scenario, the L2 can be very large SSD-backed Aerospike instance and L1 could be a small in-memory based instance. 
In this case, data is first checked in-memory, followed by L1, then L2, and then respectively synchronized from L2 to L1 and L1 to local memory. 

**Example of service with both in-memory and Aerospike cache**

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

#### Dictionary hash code

In caching mode, in order to manage cache and client/server consistency every time a model/dictionary gets re/loaded, `mly` computes a global dictionary hash code.
This hash code gets stored in the cache along with model prediction and is passed to the client in every response. 
Once a client detects a dictionary hash code difference, it automatically initiates a dictionary reload and invalidates cache entries.

## Configuration

#### Server

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
  - `Storable`: name of register storable provider
  - `Cache`: optional in-memory cache setting
      * `SizeMB`: optional cache size in MB

#### Client
 
`mly` client does not come with external config file.

To create a client, use the following snippet:
```go
mly := client.New("$modelID", []*client.Host{client.NewHost("mlServiceHost", mlServicePort)}, options ...)
```

Where optional `options` can be one of the following:
  * `NewCacheSize(sizeOption)`
  * `NewCacheScope(CacheScopeLocal|CacheScopeL1|CacheScopeL2)`
  * `NewGmetric()` - custom instance of `gmetric` service


## Usage

#### Server

To build server executable you can use the following code

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



#### Client

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

#### Output post-processing

By default, the model signature output name alongside the model prediction gets used to produce cachable output.
This process can be customized for specific needs.

The custom transformer has to use the following function signature
```go
type Transformer func(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error)
```

```go
func init() {
  transformer.Register("myTransformer", aTransformer)
}
```

Optionally you can implement a `storable` provider.

```go
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


#### Metrics

The following URIs expose metrics 
 -  `/v1/api/metric/operations` - all metrics registered in the web service
 -  `/v1/api/metric/operation/$MetricID` - individual metric `$MetricID`

#### Deployment

<a name="License"></a>

## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

<a name="Credits-and-Acknowledgements"></a>

## Contributing to `mly`

`mly` is an open source project and contributors are welcome!

See [TODO](TODO.md) list.

## Credits and Acknowledgements

**Library Author:** Adrian Witas


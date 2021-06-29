# Mly - machine learning executor on steroid for golang

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

The goal of this library to provide generic HTTP web service tensorflow ML wrapper which can speed up end to end execution by leveraging a caching. 
Caching has to be seamless, meaning client behind the scene should take care of dictionary based key generation and any model changes automatically. 

In order to get benefit of cache the model dictionary has to use enumerated layers, with keyspace providing  reasonable cache hit rate.
In practise this library can provide substantial (100x) e2e execution improvement from client side perspective, with
keyspace ranging from  millions of billions distinct keys. 

Performance transparency: each model provide both tensorflow and cache level performance metrics exposed vi REST API. 


## Introduction

This project provides library for both client and ML web service.
Service support multiple tensorflow models integration on URI level, with GET/POST method support, 
and PUT with HTTP 2.0 in full duplex mode. Service automatically detects and reloads any model changes.
Technically any HTTP client can work with the service, but to get seamless caching benefit,
it's recommended to use provide client.


To start HTTP service with 2 models, create service config.yaml file, so you can 
run GET/POST prediction using the following URL:

- Get URL ```http://mlyHost:8086/v1/api/model/$MODEL_ID/eval?modelInput1=val1&modelInputX=valueX```

@config.yaml
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

The main performance benefit comes from trading CPU with I/O, depending on keyspace the library support 3 level of caching.
In order to use cache, a client has to generate the caching key. 
Native mly client handles key generation and passing automatically.

- in process memory
- external aerospike cache
- hybrid

By default client inherit web service cache settings.

When in memory process is both client and web service manage they cache independently, meaning if data is missing in 
local client cache, client send request to ml endpoint.
When external cache is configure client first check external cache that is shared with web service, if data is found, 
it's copied to local in memory cache. Memory cache uses [scache](https://github.com/viant/scache) most recently used implementation.

To deal with large key-spaces, external cache can be further configured as L1 and L2. 
In this scenario L2 can be very large SSD backed aerospike instance and L1 could be small all 
memory based instance. In this case data is first check in memory, followed by L1 and L2, and then respectively
synchronized from L2 to L1 and L1 to local memory. 

#### Dictionary hash code.

In caching mode, in order to manage cache and client/server consistency every time model/dictionary gets re/loaded, 
mly computes global dictionary hash code. This hash code got stored in the cache alongside model prediction, 
as well its being passed to client in every response. Once client detect dict hash code difference it automatically
initiate dictionary reload and invalidates cache entries lazily.

## Configuration

#### Server

The server accept configuration with the following options
* _Models_ : list of models
  - ID: required model ID
  - URL: required model location 
        - to use s3, export AWS_SDK_LOAD_CONFIG=true
        - to use gs, export GOOGLE_APPLICATION_CREDENTIALS=true
  - Tags: optional model tags (default "serve")
  - OutputType required output type (int64, float32, etc..)
  - UseDict optional flag to use dictionary/caching (true by default)
  - KeyFields optional list of field used to generate caching key(by default all model input sorted alphabetically)
  - DataStore optional name of datastore cache
  - Transformer optional name of model output transformer
    
* _Connection_ : optional list of external aerospike connections
  - ID: required connection ID
  -  Hostnames: required aerospike hostnames


* _Datastores_ : list of datastore caches
  - ID: required datastore ID (to be matched with Model.DataStore)
  - Connection: optional connection ID
  - Namespace: optional aerospike namespace
  - Dataset: optional aerospike dataset
  - Cache: optional in memory cache setting
      * SizeMB: cache size in BM

#### Client
 
Mly client does not come with external config file

To create file use the following snippet
```go
mly := client.New("$modelID", []*client.Host{client.NewHost("mlServiceHost", mlServicePort)}, options ...)

```
where optional options can be one of the following:
  * NewCacheSize(sizeOption)
    - where sizeOption:
        * -1 - disables local and mly server cache for this client calls
        * 0 - disables local cache only
        * positive number local cache size in MB
  * NewGmetric() - custom instance of gemetric service


## Usage

#### Server



#### Client



#### Output Customization




#### Deployment



<a name="License"></a>
## License

The source code is made available under the terms of the Apache License, Version 2, as stated in the file `LICENSE`.

Individual files may be made available under their own specific license,
all compatible with Apache License, Version 2. Please see individual files for details.

<a name="Credits-and-Acknowledgements"></a>

## Contributing to mly

mly is an open source project and contributors are welcome!

See [TODO](TODO.md) list

## Credits and Acknowledgements

**Library Author:** Adrian Witas


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
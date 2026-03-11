# Configuration

## Server

See [`service/endpoint/config.go`](service/endpoint/config.go) for a more thorough description.

## Root Property `Models`

This contains a list of models (i.e. TensorFlow SavedModel package, Triton Model, or Router) that mly will set up endpoints for.

See [`service/config/model.go`](service/config/model.go) for all options.

Properties:

- `ID`: `string` - required - model ID, used to generate the URLs.
- `Debug`: `bool` - optional - enables further output and debugging.
- `URL`: `string` - required - model location source.
  * to use S3, set environment variable `AWS_SDK_LOAD_CONFIG=true`
  * to use GCS, set environment variable `GOOGLE_APPLICATION_CREDENTIALS=true`
- `Location`: `string` - optional - where a copy of the models will be stored when loading the model. Defaults to the system temporary directory.
- `Dir`: `string` - optional - any further path elements in `Location`. Mainly used if using a ZIP file with additional directories.
- `DataStore`: `string` - optional - name of Datastore to use for caching, should match `Datastores[].ID`. Server-side datastore writes are enabled only when `UseDict` is `true` or unset.
- `Transformer`: `string` - optional - name of model output transformer. See [#Transformer](#Transformer).
- `Batch`: optional - enables or overrides server-side batching configuration. See [`service/tfmodel/batcher/config/config.go`](service/tfmodel/batcher/config/config.go).
- `UseDict`: `bool` - optional - if true or unset, enables dictionary-based cache behavior, including replacing out-of-vocabulary inputs in cache keys with a special token and allowing the server to generate datastore cache entries when `DataStore` is configured. If false, the server will not generate new datastore cache entries for the model.
- `Inputs`: used to further provide or define inputs, a list of `shared.Field`. For TensorFlow models, this is automatically populated, but further caching configurations need to be specified.
  * `Name`: `string` - required - input name, only required if an entry is provided.
  * `Index`: `int` - optional - used to maintain cache key ordering.
  * `Auxiliary`: `bool` - optional - the input is permitted to be provided in an evaluation request. Mainly used for logging.
  * `Wildcard`: `bool` - conditionally required - if enabled this input will not have a vocabulary for lookup. If `UseDict` is true and `Wildcard` is false, the service will refuse to start if it cannot guess the vocabulary extraction Operation.
  * `Precision`: `int` - conditionally required - if the input is a float type and dictionary is enabled, this can be used to round the value to a lower precision which can improve cache hit rates. If `UseDict` is true, the service will refuse to start if it encounters a float input without a `Precision`.
- `KeyFields`: `[]string` - optional - list of fields used to generate caching key (by default, all model inputs, sorted alphabetically). Can be used to order and add valid inputs that can be used as a cache key but not used as prediction input.
- `Platform`: `string` - optional, defaults to `tensorflow`. Can be `tensorflow` or `triton`. See below for more information.
- `Mode`: `string` - optional, defaults to `inference`. Can be `inference` or `router`. See below for more information.
- `Auxiliary`: `[]string` - **deprecated**, optional - list of additional fields that are acceptable for eval server call. Deprecated, use `Field.Auxiliary`.
- `Outputs`: `[]shared.Field` - optional - model outputs are automatically pulled from the model. Required for Triton backends.
- `Test`: optional - enables a client request to send to self on start up.
    * `Test`: `bool` - if `true`, a client will generate a non-batch request with random values based on the model input signature.
    * `Single`: `map[string]any` - if present, will use the provided values for certain input keys, otherwise randomly generated based on model input signature.
    * `SingleBatch`: `bool` - if `true`, a client will generate a batch request with random values based on the model input signature; if `Single` is set, values will be used for provided keys.
    * `Batch`: `map[string][]any` - if present, will be used to generate a batch of requests for the self-test.

### Model Property `Platform`

The `Platform` property specifies where inference actually happens.
Currently supported values are `tensorflow` and `triton`, with `tensorflow` being the default if unspecified.

If the `Platform` is `triton`, then mly will route requests to a configured [Triton Inference Server](https://docs.nvidia.com/deeplearning/triton-inference-server/user-guide/docs/introduction/index.html).

### Model Property `Mode`

The `Mode` property enables a specific inference mode.
Currently supported values are `inference` and `router`, with `inference` being the default if unspecified.

The `inference` mode operates as expected - the inference request is processed using the model in the backend and a prediction is generated.

The `router` mode operates with a nuance - this will route input rows to a specific backend model.

Currently, only the `Platform` `triton` is supported.

## Root Property `Connections`

Can be empty - a list of external Aerospike connections.

Properties:

- `ID`: `string` - required - connection ID
- `Hostnames`: `string` - required - Aerospike hostnames

## Root Property `Datastores`

Can be empty - represent a list of caching data stores.

Properties:

- `ID`: `string` - required - datastore ID (to be matched with `Models[].DataStore`)
- `Connection`: `string` - optional - connection ID
- `Namespace`: `string` - optional - Aerospike namespace
- `Dataset`: `string` - optional - Aerospike dataset
- `Storable`: `string` - optional - name of registered `storable` provider
- `Cache`: optional - in-memory cache setting
  * `SizeMB`: `int` - optional - cache size in MB

## Root Property `Endpoint`

Contains some special administrative options.

- `Port`: `int` - optional - used in `addr` for `http.Server`, default `8080`.
- `ReadTimeoutMs`, `WriteTimeoutMs`: `int` - optional - additional settings for `http.Server`, default `5000` for both.
- `MaxHeaderBytes`: `int` - optional - additional settings for `http.Server`, default `8192` (`8 * 1024`).
- `WriteTimeout`: `int` - optional - maximum request timeout.
- `PoolMaxSize`, `BufferSize`: `int` - optional - controls implementation of `net/http/httputil`, default `512` and `131072` (`128 * 1024`), respectively.
- `MaxEvaluatorConcurrency`: `int` - optional - controls semaphore that prevents too many CGo goroutines from spawning, default `5000`.

* `EnableMemProf`: `bool` - **deprecated**, optional - enables endpoint for memory profiling - use `ProfilerPort` instead
* `EnableCPUProf`: `bool` - optional - enables endpoint for cpu profiling.
* `AllowedSubnet`: `bool` - optional - restricts administrative endpoints to IP string prefixes.
  - Restricts the system configuration, memory profile, CPU profile, and health endpoints.
* `ContinueOnRecover`: `bool` - optional - panics will not bubble up.

## Client

`mly` client does not come with an external config file.

To create a client, use the following snippet:

```go
mly := client.New("$modelID", []*client.Host{client.NewHost("mlServiceHost", mlServicePort)}, options ...)
```

Where optional `options` can be of, but not limited to, the following:
  * `WithCacheSize(sizeOption)`
  * `WithCacheScope(CacheScopeLocal|CacheScopeL1|CacheScopeL2)`
  * `WithGmetrics()` - custom instance of `gmetric` service
  * `WithHashValidation(true)` - enables client-side rejection of cached entries with a non-zero hash that differs from the client's current dictionary hash

See [`shared/client/option.go`](shared/client/option.go) for more options.

Since v0.16.0, there have been added options for Aerospike behavior.

Since each `shared/client.Service` instance is encapsulated for 1 model, there was an added option to share Aerospike connections via [`WithConnectionSharing()`](shared/client/option.go).

Additionally, Aerospike client `Policy` override options were provided.
Use `WithClientOptions()` to provide `shared/datastore/client.Option` options.
See [`shared/datastore/client/option.go`](shared/datastore/client/option.go).
Note that even if a `ClientPolicy` or `BasePolicy` is set via `shared/datastore/client.Option`, the timeout values will be applied from the server configuration unless using `WithBypassConfiguredTimeout()`.
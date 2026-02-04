# Code Architecture

mly consists of the following large conceptual (not strictly programmatically linked) steps and sub-steps:

1. Configuration processing
2. Model initialization
    1. Platform evaluator creation
    2. Caching support
3. Request handling
    1. Input processing
    2. Model inference
    3. Post-prediction processing
    4. Prediction logging
4. Model reloading

---

## 1 Configuration processing

This is a standard step in many services. This will read configuration files and populate defaults.
Details are mainly in [CONFIG.md](CONFIG.md).

*Quirk*: The configuration struct contains both the read configuration and follow-up processed configuration values. For example, `Modified` is populated during model loading, and `DictMeta` is updated when the dictionary is loaded.

---

## 2 Model Initialization

Model initialization occurs in `service.New()` which orchestrates the creation of platform-specific evaluators and supporting infrastructure.

### 2.1 Evaluator Creation

A core concept in mly is the "Evaluator."
An Evaluator is essentially something that can provide some kind of model inference.
All Evaluators implement `platform.PlatformEvaluator`:

```go
type PlatformEvaluator interface {
    Predict(ctx context.Context, params []interface{}) ([]interface{}, error)
    Signature() *domain.Signature
    Dictionary() *common.Dictionary
    Inputs() map[string]*domain.Input
    ReloadIfNeeded(ctx context.Context) error
    Close() error
}
```

There are currently 3 Evaluators:

1. TensorFlow - this Evaluator operates with a `libtensorflow` backend, and has additional logic that supports timeout-based batching.
2. Triton - this Evaluator supports sending prediction requests to a single Triton server via HTTP or gRPC.
3. Router - this Evaluator does not generate any prediction but enables rows in a prediction request to be sent to other Evaluators based on the input.

*Design issue*: Evaluator overloading and over-abstraction - the Router operates on the same interface as the TensorFlow and Triton evaluators, but vary in their behavioral labels.

### 2.2 Caching Support

Caching is implemented via `shared/datastore.Service`, which provides a multi-layer cache:

1. **Local in-memory cache** ([`scache`](https://github.com/viant/scache)): Fast local cache with TTL expiration
2. **L1 cache** (Aerospike): Primary distributed cache
3. **L2 cache** (Aerospike): Secondary distributed cache for cache warming

An important concept in mly caching is the *Dictionary hash*.
This is stored with cached values, and is intended to invalidate entries when the model changes (e.g., there is a model weights update).

*Quirk*: Client-read, server-write - based on the observation that if a client does not find a cache entry, then the server is unlikely to also find a cache entry, and to skip the latency overhead from a remote cache check, the server does not check for a cache entry.

*Design debt*: The current client-read, server-write introduces a case when multiple clients concurrently find that a cache entry is missing, and sends the same request to potentially the same mly server, causing the same server to run the same prediction multiple times. This should be controlled on the server side, to avoid unnecessary compute.

*Design debt*: Aerospike coupling - the current implementation depends on Aerospike constructs.

---

## 3 Request Handling

The mly service occupies most of its lifetime serving this purpose. Currently, mly is designed to focus around HTTP requests, using HTTP/2.

Data flow:

```
HTTP Request
→ service.Handler.ServeHTTP()
→ service.Service.Do()
→ service.Evaluator.Predict()
→ service.domain.Transformer
→ service.Response
```

### 3.1 Input Processing

The Input processing step is primarily focused around logic of pulling data from an HTTP-compliant, JSON or URL-based payload and pushing it into a Go (and CGo) compatible data structure for model inference.

This step revolves mainly around the `service/request.Request` struct.

Key components:
- **Feeds**: `[]interface{}` shaped as `[numInputs]([batchSize][1]T)` for model consumption
- **Input**: `*transfer.Input` for Transformer support

The `UnmarshalJSONObject()` method implements `gojay.UnmarshalerJSONObject` for high-performance JSON parsing.

The interaction with *Model inference* involves the `Feeds` field.

*Quirk*: client batching payload - mly provides a convenience / payload reduction feature that permits payloads to have both inputs with a list of 1 values as well as inputs with a list of batch size of values. The server will expand the payload to fit the expected batch size times inputs matrix for the Evaluators.

*Quirk*: payload reading order - the JSON payload must have the `batch_size` key existing before other input keys, as that is required to know if the parser should be expecting a list of values or scalar values.

*Design debt*: `Feeds` type - most of the requests are tracked via input names than offsets; the intermediate data form should be a `map[string]interface{}` (or even `map[string][]interface{}` to capture a potential batch layer), and the conversion to an offset-based slice should be isolated to TensorFlow graph related code.

### 3.2 Model Inference

Model inference is delegated to the platform-specific evaluator via `Predict()`:

**TensorFlow** (`service/tfmodel`):
- Optional batching via `service/tfmodel/batcher.Service` aggregates concurrent requests
- Direct evaluation via `service/tfmodel/evaluator.Service` runs TensorFlow session
- Semaphore-controlled concurrency prevents overload

For Triton, [concurrency and timeout-based batching is controlled via Triton](https://docs.nvidia.com/deeplearning/triton-inference-server/user-guide/docs/user_guide/model_configuration.html).

**Triton** (`service/triton`):
- HTTP or gRPC call to Triton Inference Server
- Input tensors serialized per Triton protocol
- Timeout-controlled requests

The router is a control layer that can route to various model inference services.
Each downstream evaluator is responsible for controlling their own lifetime and batching concerns.
In theory, a router can also route to another router, but no such implementation yet exists.

**Router** (`service/platform/router`):
- Extracts routing key from input
- Groups rows by target model
- Parallel dispatch to downstream evaluators
- Result reassembly in original order

### 3.3 Post-Prediction Processing

After inference, `Service.buildResponse()` handles output transformation:

**Transformer execution**
The configured `domain.Transformer` function transforms raw model output into a `common.Storable` for serialization.
The default transformer extracts values keyed by output tensor names.

The `domain.Transformer` signature:

```go
type Transformer func(ctx context.Context, signature *Signature, input *gtly.Object, output interface{}) (common.Storable, error)
```

**Cache storage**
If caching is enabled, transformed results are stored asynchronously via `datastore.Put()`.

*Design debt*: Batch-based Transformer - the current Transformer API operates at the request level outputs but at row-level inputs, and is invoked per row.

### 3.4 Prediction Logging

If `Stream` is configured, the `stream.Service` logs requests for analytics:
- Request body
- Model output
- Inference duration

Logging uses `github.com/viant/tapper` for configurable output destinations.

---

## 4 Model Reloading

Model reloading runs continuously in a background goroutine (`Service.pollModelReload()`), checking for updates at configurable intervals (`ReloadPollIntervalSeconds`).

The `ReloadIfNeeded()` implementation is platform-specific, and varies similarly to model prediction in how much is implemented vs. delegated:

**TensorFlow** (`service/tfmodel.Service`):
1. Check file modification times at `URL`
2. If changed: copy model files to `Location`, load SavedModel
3. Extract signature and dictionary from graph
4. Create new `service/tfmodel/evaluator.Service` and optionally `service/tfmodel/batcher.Service`
5. Atomically swap Evaluators under mutex protection

**Triton** (`service/triton.TritonEvaluator`):
1. Check model health via `ModelReady()` API
2. If not ready and in EXPLICIT mode: call `ModelLoad()`
3. Refresh metadata via `ModelMetadata()` if signature not yet captured

**Router** (`service/platform/router.Router`):
1. Check routing configuration file modification
2. Reload routing table if changed
3. Create/destroy downstream Evaluators as needed
4. Atomically swap routing table under mutex protection
5. Unload unused models from Triton via Model Control API

Reload health is tracked via `Service.ReloadOK` for centralized health reporting.

*Design issue*: Over-abstraction of `ReloadIfNeeded()` - we note that this is a very high-level abstraction that could be broken down into separate concerns e.g., check health, load model, check if reload needed, etc.
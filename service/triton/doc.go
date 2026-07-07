package triton

/*
Package triton integrates mly with a Triton Inference Server over the
KServe/Open-Inference v2 protocol (gRPC preferred; the HTTP client is deprecated).

# Output ordering contract

The v2 protocol does not guarantee that the order of tensors in a ModelInfer
response matches the order of outputs reported by ModelMetadata. The only
response-side ordering guarantee is that raw_output_contents[i] aligns with
outputs[i]; the outputs list itself may be emitted in any order. In practice,
models exported with non-deterministic tensor ordering can return inference
outputs in a different order than their own metadata declares.

Because every response tensor carries its own name, outputs are addressed by
name rather than by position:

  - TritonClient.ModelInfer returns outputs as map[string]interface{} keyed by
    the Triton output tensor name. The map deliberately carries no ordering, so
    no caller can depend on the order Triton happened to return tensors in.

  - TritonEvaluator.Predict is the single place that establishes order: it maps
    the named results into signature.Outputs order (the order captured from
    ModelMetadata at reload) and returns a []interface{} that satisfies the
    platform.Predictor contract result[i] == signature.Outputs[i].

Consumers — including the router, which reassembles outputs from multiple models
into one response — must rely on this evaluator-established order and must not
assume the raw ModelInfer response order matches metadata. When outputs share a
datatype (e.g. two FP32 scalars), a positional mismatch is otherwise silently
undetectable: values are mislabeled with no type error.
*/

# Notes about Incoherent or Inconsistent Architecture and Design

## 1 `service/domain`

`service/domain.Input` and `service/domain.Ouput` have dependencies on TensorFlow but are part of the generalized `service/platform.PlatformEvaluator` interface.
This is an unnecessary coupling and should be removed.
The usage in the `service` module is mainly for type mapping from JSON to Go to Go-type for TensorFlow.


## 2 `service/request.Request.Feeds`

This seems too tailored towards TensorFlow inference.
A common operation in Triton is to convert the offset-based slice data back into name-based slice data.
Might see cognitive improvement if we reduce that back-and-forth.

## 3 Input Validation

Incorrect batch size results in panic.


## 4 Triton Management Frequency

Higher frequency model load frequency.
Mainly useful during development, but may be useful in production environments where Triton server dies or is restarted?
No, seems like that would be a separate issue in it of itself.
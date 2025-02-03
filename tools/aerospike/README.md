# Aerospike Connection Tool

Tooling used to help with observing effects of https://github.com/viant/mly/issues/37.

Create an mly server with `slf` and appropriate Aerospike instances running.

Run `go run ./tools/aerospike`.

Use `asadm` with `watch 1 info network` to monitor connections.

# Backwards compatibility e2e testing

1. Find latest tagged revision(s). 
  - Should also be parameterizable.
  - If multiple occur, select largest. 
  - TODO: major version backwards compatibility matrix.
  - If the current commit is the the latest tagged revision, search for an older tagged revision.
2. Create build using tagged revision. Run using tagged revision's configuration.
3. Create build using latest revision. Run using latest revision's configuration.
4. Run e2e tests using a 2x2 (until full matrix) of tagged client, latest client, tagged server, latest server.
  - Need monitoring of cache thrashing.
  - Need to ensure no strict backwards compatibility errors. 
    - Currently the only dedictated API will be for the HTTP endpoints.
    - Since as of v0.8.3 we don't leverage [`internal`](https://go.dev/doc/go1.4#internalpackages), we will mark all available Golang API as unstable.


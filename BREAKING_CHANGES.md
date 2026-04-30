# Breaking Changes

See [wiki](https://github.com/viant/mly/wiki) for older entries and non-breaking changes.

## `v0.19.x` to `v0.20.x`

1. `example/server.RunApp` return changes from none to `error`.
2. `service.New()` added parameter `tritonClients`.
3. `service.NewWithPlatform()` removed.
3. `service/endpoint/checker.SelfTest()` removed parameters `inputs_` and `outputs`.
4. `service/endpoint/health.(*HealthHandler).RegisterHealthPoint()` second parameter changed.
5. `service/endpoint/prometheus.Handler()` parameter changed.
6. `service/endpoint.Build()` added parameter `tritonClients`.

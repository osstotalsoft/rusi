
## Tracing config
* `collectorEnpoint` (string) the Telemetry Collector Endpoint.
* `propagator` (string) w3c | jaeger

Example:

```yaml
  telemetry:
    collectorEnpoint: opentelemetry-opentelemetry-collector.global-infra:4317
    tracing:
      propagator: w3c # w3c | jaeger
```
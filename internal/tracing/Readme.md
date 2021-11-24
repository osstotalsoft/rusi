
## Jaeger config
* `useAgent` (bool) set to true if using a Jaeger Agent
* `collectorEndpointAddress` the Jaeger Collector Endpoint. Set only if `useAgent` is false, to send the tracing data directly to the collector.

Example:

```yaml
  tracing:
    samplingRate: '1'
    jaeger:
      useAgent: false
      collectorEndpointAddress: 'http://kube-worker1.totalsoft.local:31034/api/traces'

```

## Development environment
On the development environment we can configure Jaeger to send data directly to the collector.

Alternatively we can install a jaeger agent on the local machine:

```cmd
docker pull jaegertracing/jaeger-agent:1.12.0

docker run -it -p 6831:6831/udp jaegertracing/jaeger-agent:1.12.0 --collector.host-port='kube-worker1.totalsoft.local:31335'
```
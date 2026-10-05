rusi
===============
>Runtime Sidecar - a [Dapr](https://github.com/dapr/dapr) inspired story

![rusi](assets/logo.png)

## Contributing on windows

- Install Go: 
  - Download Go for windows; Go 1.17
- Editor: 
  - Visual studio Code + Go extension
- Build: 
    - install make
        ```shell
        choco install make
        ```
    - install protoc
        ```shell
        choco install protoc
        ```
    - build with make
        ```shell
        make build-linux
        ```
- Debug: 
    - install delve debug tool: dlv-dap - suggested by vs-code
    - debug with vscode :> Run sidecar
- Test:
    - install gcc
        ```shell
        choco install mingw
        ```
    - test with make
        ```shell
        make test
        ```
	
## Usage

### Configure Components

In self-hosted mode, component files have to be saved on the local machine, and provide a `components-path` to the sidecar.
In Kubernetes mode Rusi will query the kubernetes api in order to find and register all components   

1. Create a pub/sub message broker component 
    ```yaml
    apiVersion: rusi.io/v1alpha1
    kind: Component
    metadata:
      name: natsstreaming-pubsub
    spec:
      type: pubsub.natsstreaming
      version: v1
      metadata:
      - name: natsURL
        value: "replace with your host"
      - name: natsStreamingClusterID
        value: "replace with your cluster name"
        # below are subscription configuration.
      - name: subscriptionType
        value: queue # Required. Allowed values: topic, queue.
      - name: ackWaitTime
        value: "" # Optional.
      - name: maxInFlight
        value: "1" # Optional.
      - name: durableSubscriptionName
        value: "" # Optional.
    ```

    Kafka is also available (`pubsub.kafka`, see `examples/components/comp-kafka.yaml`):
    ```yaml
    spec:
      type: pubsub.kafka
      version: v1
      metadata:
      - name: bootstrapServers
        value: "broker1:9092,broker2:9092" # Required.
      - name: groupId
        value: "" # Optional. Defaults to the app id.
    ```
    Subscription options: `qGroup` (default true) shares the consumer group; false gives each subscriber its own group.
    `deliverNewMessagesOnly` (default true) picks the start offset for a new group (newest vs oldest).
    Each partition is processed sequentially and its offset is committed after the handler returns, including on handler error
    (Kafka has no per-message redelivery). `durable`, `maxConcurrentMessages` and `ackWaitTime` are ignored.
   
2. Add custom middlewares (optional)
    ```yaml
    apiVersion: rusi.io/v1alpha1
    kind: Component
    metadata:
      name: uppercase
    spec:
      type: middleware.http.uppercase
      version: v1
    ```
3. Configure middleware pipeline (optional)

    configuration is not mandatory, unless you want to specify a specific pipeline for pubsub
   ```yaml
    apiVersion: rusi.io/v1alpha1
    kind: Configuration
    metadata:
      name: node-pipeline-config
    spec:
      subscriberPipeline:
        handlers:
        - name: pubsub-uppercase
          type: middleware.pubsub.uppercase
          # add other middlewares
      publisherPipeline:
        handlers:
          - name: pubsub-uppercase
            type: middleware.pubsub.uppercase
          # add other middlewares
    ```
### run rusid 
 - kubernetes 
```shell
go run cmd/rusid/sidecar.go --mode kubernetes --app-id your-app-id --config "kube-config-resource-name"
```
If you run out of cluster, it will use your kubectl current context, to scan for all components in all namespaces. 
 - standalone
```shell
go run cmd/rusid/sidecar.go --app-id your-app-id --components-path="path-to-components-folder" --config "path-to-config.yaml"
```

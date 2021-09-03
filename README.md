# rusi
Rusi is a Runtime Sidecar and a [Dapr](https://github.com/dapr/dapr) inspired story

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
- Run:
    ```shell
    go run cmd/rusid/sidecar.go --app-id node-subscriber --config "config-node-pipeline.yaml"
    ```
  - kubernetes mode:
    - uses kubectl default context
    - looks for configuration resources in all namespaces
    
  - local mode: tbd
	



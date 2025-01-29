## Prerequisites
Use **Windows Subsystem for Linux** (Ubuntu)

1. Install go v1.17 from [here](https://golang.org/doc/install)

2. Install make 
```bash
apt install make
```
1. Install the kubernetes code generator:
```bash
go install k8s.io/code-generator@v0.22.2
sudo chmod 777 ~/go/pkg/mod/k8s.io/code-generator@v0.22.2/generate-groups.sh
```


## Generate client
Go to rusi folder (repository root) and run the following make task:
```bash
make generate-crd
```
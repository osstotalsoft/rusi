#CODE_GENERATOR_DIR?=/d/GoProjects/pkg/mod/k8s.io/code-generator\@v0.22.2
CODE_GENERATOR_DIR?=~/go/pkg/mod/k8s.io/code-generator@v0.22.2
TOOLS_DIR?=pkg/operator/tools

generate-apis:
	$(CODE_GENERATOR_DIR)/generate-groups.sh all rusi/pkg/operator/client rusi/pkg/operator/apis \
    	  rusi:v1alpha1 \
    	  -h $(TOOLS_DIR)/boilerplate.go.txt

	cp -rf ./rusi/pkg/ ./
	rm -rf ./rusi/

	#$(TOOLS_DIR)/update-codegen.sh

generate-crd:
	controller-gen crd paths="rusi/pkg/operator/apis/..." +output:dir=helm/crds
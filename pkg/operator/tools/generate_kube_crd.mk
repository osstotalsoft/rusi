
#CODE_GENERATOR_DIR?=/d/GoProjects/pkg/mod/k8s.io/code-generator\@v0.22.1
CODE_GENERATOR_DIR?=~/go/pkg/mod/k8s.io/code-generator@v0.22.1
TOOLS_DIR?=pkg/operator/tools

generate-crd:
	$(CODE_GENERATOR_DIR)/generate-groups.sh all rusi/pkg/operator/client rusi/pkg/operator/apis \
    	  "components:v1alpha1 configuration:v1alpha1 subscriptions:v1alpha1" \
    	  -h $(TOOLS_DIR)/boilerplate.go.txt

	#$(TOOLS_DIR)/update-codegen.sh

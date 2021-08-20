
TOOLS_DIR?=./pkg/operator/tools

generate-crd:
	/d/GoProjects/pkg/mod/k8s.io/code-generator\@v0.22.1/generate-groups.sh all \
 			rusi/pkg/operator/client rusi/pkg/operator/apis "components:v1alpha1" \
 			-h $(TOOLS_DIR)/boilerplate.go.txt
	#$(TOOLS_DIR)/update-codegen.sh
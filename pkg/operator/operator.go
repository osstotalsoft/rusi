package operator

import (
	"fmt"
	"google.golang.org/grpc"
	"k8s.io/klog/v2"
	"net"
	"rusi/pkg/kube"
	"rusi/pkg/operator/client/clientset/versioned"
	operatorv1 "rusi/pkg/proto/operator/v1"
)

const serverPort = 6500

func Run() {
	s := grpc.NewServer()

	cfg := kube.GetConfig()
	client, _ := versioned.NewForConfig(cfg)

	operatorv1.RegisterRusiOperatorServer(s, &operatorServer{client})

	lis, err := net.Listen("tcp", fmt.Sprintf(":%v", serverPort))
	if err != nil {
		klog.Fatalf("error starting tcp listener: %v", err)
	}
	klog.Info("starting Operator gRPC server")
	if err := s.Serve(lis); err != nil {
		klog.Fatalf("gRPC server error: %v", err)
	}
}

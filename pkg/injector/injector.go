package injector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
	"net/http"
	"rusi/pkg/utils"
	"time"

	v1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	"k8s.io/client-go/kubernetes"
)

const (
	port                                      = 4000
	getKubernetesServiceAccountTimeoutSeconds = 10
)

var allowedControllersServiceAccounts = []string{
	"replicaset-controller",
	"deployment-controller",
	"cronjob-controller",
	"job-controller",
	"statefulset-controller",
	"daemon-set-controller",
}

type Injector interface {
	Run(ctx context.Context) error
}

type injector struct {
	config       Config
	deserializer runtime.Decoder
	server       *http.Server
	kubeClient   *kubernetes.Clientset
	authUIDs     []string
}

// NewInjector returns a new Injector instance with the given config.
func NewInjector(authUIDs []string, config Config, kubeClient *kubernetes.Clientset) Injector {
	mux := http.NewServeMux()

	i := &injector{
		config: config,
		deserializer: serializer.NewCodecFactory(
			runtime.NewScheme(),
		).UniversalDeserializer(),
		server: &http.Server{
			Addr:    fmt.Sprintf(":%d", port),
			Handler: mux,
		},
		kubeClient: kubeClient,
		authUIDs:   authUIDs,
	}

	mux.HandleFunc("/mutate", i.handleRequest)
	return i
}

func (i *injector) Run(ctx context.Context) error {
	klog.InfoS("Starting Rusi Injector server at port 4000")
	return i.server.ListenAndServeTLS(i.config.TLSCertFile, i.config.TLSKeyFile)
}

func (i *injector) handleRequest(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	var body []byte
	if r.Body != nil {
		if data, err := ioutil.ReadAll(r.Body); err == nil {
			body = data
		}
	}
	if len(body) == 0 {
		klog.Error("empty body")
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	if contentType != runtime.ContentTypeJSON {
		klog.Errorf("Content-Type=%s, expect %s", contentType, runtime.ContentTypeJSON)
		http.Error(
			w,
			fmt.Sprintf("invalid Content-Type, expect `%s`", runtime.ContentTypeJSON),
			http.StatusUnsupportedMediaType,
		)

		return
	}

	var admissionResponse *v1.AdmissionResponse
	var patchOps []PatchOperation
	var err error

	ar := v1.AdmissionReview{}
	_, gvk, err := i.deserializer.Decode(body, nil, &ar)
	if err != nil {
		klog.Errorf("Can't decode body: %v", err)
	} else {
		if i.config.ValidateServiceAccount && !utils.StringSliceContains(ar.Request.UserInfo.UID, i.authUIDs) {
			err = errors.New(fmt.Sprintf("service account '%s' not on the list of allowed controller accounts", ar.Request.UserInfo.Username))
			klog.Error(err)
		} else if ar.Request.Kind.Kind != "Pod" {
			err = errors.New(fmt.Sprintf("invalid kind for review: %s", ar.Kind))
			klog.Error(err)
		} else {
			patchOps, err = i.getPodPatchOperations(&ar, i.config.Namespace, i.config.SidecarImage,
				i.config.SidecarImagePullPolicy, i.kubeClient)
		}
	}

	diagAppID := getAppIDFromRequest(ar.Request)

	if err != nil {
		admissionResponse = toAdmissionResponse(err)
		klog.Errorf("Sidecar injector failed to inject for app '%s'. Error: %s", diagAppID, err)
	} else if len(patchOps) == 0 {
		admissionResponse = &v1.AdmissionResponse{
			Allowed: true,
		}
	} else {
		var patchBytes []byte
		patchBytes, err = json.Marshal(patchOps)
		if err != nil {
			admissionResponse = toAdmissionResponse(err)
		} else {
			admissionResponse = &v1.AdmissionResponse{
				Allowed: true,
				Patch:   patchBytes,
				PatchType: func() *v1.PatchType {
					pt := v1.PatchTypeJSONPatch
					return &pt
				}(),
			}
		}
	}

	admissionReview := v1.AdmissionReview{}
	if admissionResponse != nil {
		admissionReview.Response = admissionResponse
		if ar.Request != nil {
			admissionReview.Response.UID = ar.Request.UID
			admissionReview.SetGroupVersionKind(*gvk)
		}
	}

	respBytes, err := json.Marshal(admissionReview)
	if err != nil {
		http.Error(
			w,
			err.Error(),
			http.StatusInternalServerError,
		)

		klog.Errorf("Injector failed to inject for app '%s'. Can't deserialize response: %s", diagAppID, err)
	}
	w.Header().Set("Content-Type", runtime.ContentTypeJSON)
	if _, err := w.Write(respBytes); err != nil {
		klog.Error(err)
	} else {
		if len(patchOps) == 0 {
			klog.Infof("Injector ignored app '%s'", diagAppID)
		} else {
			klog.Infof("Injector succeeded injection for app '%s'", diagAppID)
		}
	}
}

// AllowedControllersServiceAccountUID returns an array of UID, list of allowed service account on the webhook handler.
func AllowedControllersServiceAccountUID(ctx context.Context, kubeClient *kubernetes.Clientset) ([]string, error) {
	allowedUids := []string{}
	for i, allowedControllersServiceAccount := range allowedControllersServiceAccounts {
		saUUID, err := getServiceAccount(ctx, kubeClient, allowedControllersServiceAccount)
		// i == 0 => "replicaset-controller" is the only one mandatory
		if err != nil {
			if i == 0 {
				return nil, err
			}
			klog.Warningf("Unable to get SA %s UID (%s)", allowedControllersServiceAccount, err)
			continue
		}
		allowedUids = append(allowedUids, saUUID)
	}

	return allowedUids, nil
}

func getServiceAccount(ctx context.Context, kubeClient *kubernetes.Clientset, allowedControllersServiceAccount string) (string, error) {
	ctxWithTimeout, cancel := context.WithTimeout(ctx, getKubernetesServiceAccountTimeoutSeconds*time.Second)
	defer cancel()

	sa, err := kubeClient.CoreV1().ServiceAccounts(metav1.NamespaceSystem).Get(ctxWithTimeout, allowedControllersServiceAccount, metav1.GetOptions{})
	if err != nil {
		return "", err
	}

	return string(sa.ObjectMeta.UID), nil
}

func toAdmissionResponse(err error) *v1.AdmissionResponse {
	return &v1.AdmissionResponse{
		Result: &metav1.Status{
			Message: err.Error(),
		},
	}
}

func getAppIDFromRequest(req *v1.AdmissionRequest) string {
	// default App ID
	appID := ""

	// if req is not given
	if req == nil {
		return appID
	}

	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		klog.Warningf("could not unmarshal raw object: %v", err)
	} else {
		appID = getAppID(pod)
	}

	return appID
}

package injector

import (
	"encoding/json"
	"fmt"
	"path"
	"rusi/pkg/utils"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"k8s.io/klog/v2"

	v1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
)

const (
	sidecarContainerName           = "rusid"
	rusiEnabledKey                 = "rusi.io/enabled"
	rusiConfigKey                  = "rusi.io/config"
	appIDKey                       = "rusi.io/app-id"
	rusiLogLevel                   = "rusi.io/log-level"
	rusiEnableMetricsKey           = "rusi.io/enable-metrics"
	rusiMetricsPortKey             = "rusi.io/metrics-port"
	rusiEnableDebugKey             = "rusi.io/enable-debug"
	rusiDebugPortKey               = "rusi.io/debug-port"
	rusiEnvKey                     = "rusi.io/env"
	rusiCPULimitKey                = "rusi.io/sidecar-cpu-limit"
	rusiMemoryLimitKey             = "rusi.io/sidecar-memory-limit"
	rusiCPURequestKey              = "rusi.io/sidecar-cpu-request"
	rusiMemoryRequestKey           = "rusi.io/sidecar-memory-request"
	rusiLivenessProbeDelayKey      = "rusi.io/sidecar-liveness-probe-delay-seconds"
	rusiLivenessProbeTimeoutKey    = "rusi.io/sidecar-liveness-probe-timeout-seconds"
	rusiLivenessProbePeriodKey     = "rusi.io/sidecar-liveness-probe-period-seconds"
	rusiLivenessProbeThresholdKey  = "rusi.io/sidecar-liveness-probe-threshold"
	rusiReadinessProbeDelayKey     = "rusi.io/sidecar-readiness-probe-delay-seconds"
	rusiReadinessProbeTimeoutKey   = "rusi.io/sidecar-readiness-probe-timeout-seconds"
	rusiReadinessProbePeriodKey    = "rusi.io/sidecar-readiness-probe-period-seconds"
	rusiReadinessProbeThresholdKey = "rusi.io/sidecar-readiness-probe-threshold"

	containersPath                = "/spec/containers"
	sidecarAPIGRPCPort            = 50003
	userContainerRusiGRPCPortName = "RUSI_GRPC_PORT"
	sidecarGRPCPortName           = "rusi-grpc"
	sidecarDiagnosticsPortName    = "rusi-diag"
	sidecarDebugPortName          = "rusi-debug"
	defaultLogLevel               = "2"
	apiAddress                    = "rusi-api"
	apiPort                       = 80
	kubernetesMountPath           = "/var/run/secrets/kubernetes.io/serviceaccount"
	defaultConfig                 = "default"
	defaultEnabledMetric          = true
	defaultMetricsPort            = 9090
	defaultSidecarDebug           = false
	defaultSidecarDebugPort       = 40000

	sidecarDiagnosticsPort            = 8080
	sidecarHealthzPath                = "healthz"
	defaultHealthzProbeDelaySeconds   = 3
	defaultHealthzProbeTimeoutSeconds = 3
	defaultHealthzProbePeriodSeconds  = 6
	defaultHealthzProbeThreshold      = 3
	trueString                        = "true"
)

// PatchOperation represents a discreet change to be applied to a Kubernetes resource.
type PatchOperation struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value,omitempty"`
}

func (i *injector) getPodPatchOperations(ar *v1.AdmissionReview,
	namespace, image, imagePullPolicy string,
	kubeClient *kubernetes.Clientset) ([]PatchOperation, error) {

	req := ar.Request
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		errors.Wrap(err, "could not unmarshal raw object")
		return nil, err
	}

	klog.Infof(
		"AdmissionReview for Kind=%v, Namespace=%v Name=%v (%v) UID=%v "+
			"patchOperation=%v UserInfo=%v",
		req.Kind,
		req.Namespace,
		req.Name,
		pod.Name,
		req.UID,
		req.Operation,
		req.UserInfo,
	)

	if !isResourceRusiEnabled(pod.Annotations) || podContainsSidecarContainer(&pod) {
		return nil, nil
	}

	id := getAppID(pod)
	err := ValidateKubernetesAppID(id)
	if err != nil {
		return nil, err
	}

	apiSvcAddress := getServiceAddress(apiAddress, namespace, i.config.KubeClusterDomain, apiPort)

	tokenMount := getTokenVolumeMount(pod)
	sidecarContainer, err := getSidecarContainer(pod.Annotations, id, image, imagePullPolicy, req.Namespace,
		apiSvcAddress, tokenMount)
	if err != nil {
		return nil, err
	}

	patchOps := []PatchOperation{}
	envPatchOps := []PatchOperation{}
	var path string
	var value interface{}
	if len(pod.Spec.Containers) == 0 {
		path = containersPath
		value = []corev1.Container{*sidecarContainer}
	} else {
		envPatchOps = addRusiEnvVarsToContainers(pod.Spec.Containers)
		path = "/spec/containers/-"
		value = sidecarContainer
	}

	patchOps = append(
		patchOps,
		PatchOperation{
			Op:    "add",
			Path:  path,
			Value: value,
		},
	)
	patchOps = append(patchOps, envPatchOps...)

	return patchOps, nil
}

func getServiceAddress(name, namespace, clusterDomain string, port int) string {
	return fmt.Sprintf("%s.%s.svc.%s:%d", name, namespace, clusterDomain, port)
}

// This function add Rusi environment variables to all the containers in any Rusi enabled pod.
// The containers can be injected or user defined.
func addRusiEnvVarsToContainers(containers []corev1.Container) []PatchOperation {
	portEnv := []corev1.EnvVar{
		{
			Name:  userContainerRusiGRPCPortName,
			Value: strconv.Itoa(sidecarAPIGRPCPort),
		},
	}
	envPatchOps := make([]PatchOperation, 0, len(containers))
	for i, container := range containers {
		path := fmt.Sprintf("%s/%d/env", containersPath, i)
		patchOps := getEnvPatchOperations(container.Env, portEnv, path)
		envPatchOps = append(envPatchOps, patchOps...)
	}
	return envPatchOps
}

// This function only add new environment variables if they do not exist.
// It does not override existing values for those variables if they have been defined already.
func getEnvPatchOperations(envs []corev1.EnvVar, addEnv []corev1.EnvVar, path string) []PatchOperation {
	if len(envs) == 0 {
		// If there are no environment variables defined in the container, we initialize a slice of environment vars.
		return []PatchOperation{
			{
				Op:    "add",
				Path:  path,
				Value: addEnv,
			},
		}
	}
	// If there are existing env vars, then we are adding to an existing slice of env vars.
	path += "/-"

	var patchOps []PatchOperation
LoopEnv:
	for _, env := range addEnv {
		for _, actual := range envs {
			if actual.Name == env.Name {
				// Add only env vars that do not conflict with existing user defined/injected env vars.
				continue LoopEnv
			}
		}
		patchOps = append(patchOps, PatchOperation{
			Op:    "add",
			Path:  path,
			Value: env,
		})
	}
	return patchOps
}

func getTokenVolumeMount(pod corev1.Pod) *corev1.VolumeMount {
	for _, c := range pod.Spec.Containers {
		for _, v := range c.VolumeMounts {
			if v.MountPath == kubernetesMountPath {
				return &v
			}
		}
	}
	return nil
}

func podContainsSidecarContainer(pod *corev1.Pod) bool {
	for _, c := range pod.Spec.Containers {
		if c.Name == sidecarContainerName {
			return true
		}
	}
	return false
}

func getConfig(annotations map[string]string) string {
	return getStringAnnotationOrDefault(annotations, rusiConfigKey, defaultConfig)
}

func getEnableDebug(annotations map[string]string) bool {
	return getBoolAnnotationOrDefault(annotations, rusiEnableDebugKey, defaultSidecarDebug)
}

func getDebugPort(annotations map[string]string) int {
	return int(getInt32AnnotationOrDefault(annotations, rusiDebugPortKey, defaultSidecarDebugPort))
}

func getAppID(pod corev1.Pod) string {
	return getStringAnnotationOrDefault(pod.Annotations, appIDKey, pod.GetName())
}

func getLogLevel(annotations map[string]string) string {
	return getStringAnnotationOrDefault(annotations, rusiLogLevel, defaultLogLevel)
}

func getBoolAnnotationOrDefault(annotations map[string]string, key string, defaultValue bool) bool {
	enabled, ok := annotations[key]
	if !ok {
		return defaultValue
	}
	s := strings.ToLower(enabled)
	// trueString is used to silence a lint error.
	return (s == "y") || (s == "yes") || (s == trueString) || (s == "on") || (s == "1")
}

func getStringAnnotationOrDefault(annotations map[string]string, key, defaultValue string) string {
	if val, ok := annotations[key]; ok && val != "" {
		return val
	}
	return defaultValue
}

func getStringAnnotation(annotations map[string]string, key string) string {
	return annotations[key]
}

func getInt32AnnotationOrDefault(annotations map[string]string, key string, defaultValue int) int32 {
	s, ok := annotations[key]
	if !ok {
		return int32(defaultValue)
	}
	value, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return int32(defaultValue)
	}
	return int32(value)
}

func getProbeHTTPHandler(port int32, pathElements ...string) corev1.Handler {
	return corev1.Handler{
		HTTPGet: &corev1.HTTPGetAction{
			Path: formatProbePath(pathElements...),
			Port: intstr.IntOrString{IntVal: port},
		},
	}
}

func formatProbePath(elements ...string) string {
	pathStr := path.Join(elements...)
	if !strings.HasPrefix(pathStr, "/") {
		pathStr = fmt.Sprintf("/%s", pathStr)
	}
	return pathStr
}

func appendQuantityToResourceList(quantity string, resourceName corev1.ResourceName, resourceList corev1.ResourceList) (*corev1.ResourceList, error) {
	q, err := resource.ParseQuantity(quantity)
	if err != nil {
		return nil, err
	}
	resourceList[resourceName] = q
	return &resourceList, nil
}

func getResourceRequirements(annotations map[string]string) (*corev1.ResourceRequirements, error) {
	r := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}
	cpuLimit, ok := annotations[rusiCPULimitKey]
	if ok {
		list, err := appendQuantityToResourceList(cpuLimit, corev1.ResourceCPU, r.Limits)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing sidecar cpu limit")
		}
		r.Limits = *list
	}
	memLimit, ok := annotations[rusiMemoryLimitKey]
	if ok {
		list, err := appendQuantityToResourceList(memLimit, corev1.ResourceMemory, r.Limits)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing sidecar memory limit")
		}
		r.Limits = *list
	}
	cpuRequest, ok := annotations[rusiCPURequestKey]
	if ok {
		list, err := appendQuantityToResourceList(cpuRequest, corev1.ResourceCPU, r.Requests)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing sidecar cpu request")
		}
		r.Requests = *list
	}
	memRequest, ok := annotations[rusiMemoryRequestKey]
	if ok {
		list, err := appendQuantityToResourceList(memRequest, corev1.ResourceMemory, r.Requests)
		if err != nil {
			return nil, errors.Wrap(err, "error parsing sidecar memory request")
		}
		r.Requests = *list
	}

	if len(r.Limits) > 0 || len(r.Requests) > 0 {
		return &r, nil
	}
	return nil, nil
}

func getEnableMetrics(annotations map[string]string) bool {
	return getBoolAnnotationOrDefault(annotations, rusiEnableMetricsKey, defaultEnabledMetric)
}

func getMetricsPort(annotations map[string]string) int {
	return int(getInt32AnnotationOrDefault(annotations, rusiMetricsPortKey, defaultMetricsPort))
}

func isResourceRusiEnabled(annotations map[string]string) bool {
	return getBoolAnnotationOrDefault(annotations, rusiEnabledKey, false)
}

func getPullPolicy(pullPolicy string) corev1.PullPolicy {
	switch pullPolicy {
	case "Always":
		return corev1.PullAlways
	case "Never":
		return corev1.PullNever
	case "IfNotPresent":
		return corev1.PullIfNotPresent
	default:
		return corev1.PullIfNotPresent
	}
}

func getSidecarContainer(annotations map[string]string, id, rusiSidecarImage, imagePullPolicy,
	namespace, controlPlaneAddress string, tokenVolumeMount *corev1.VolumeMount) (*corev1.Container, error) {

	metricsEnabled := getEnableMetrics(annotations)
	pullPolicy := getPullPolicy(imagePullPolicy)
	httpHandler := getProbeHTTPHandler(sidecarDiagnosticsPort, sidecarHealthzPath)

	allowPrivilegeEscalation := false

	ports := []corev1.ContainerPort{
		{
			ContainerPort: sidecarAPIGRPCPort,
			Name:          sidecarGRPCPortName,
		},
		{
			ContainerPort: sidecarDiagnosticsPort,
			Name:          sidecarDiagnosticsPortName,
		},
	}

	cmd := []string{"/rusid"}

	args := []string{
		"--rusi-grpc-port", fmt.Sprintf("%v", sidecarAPIGRPCPort),
		"--app-id", id,
		"--mode", "kubernetes",
		"--v", getLogLevel(annotations),
		"--control-plane-address", controlPlaneAddress,
		"--config", getConfig(annotations),
		fmt.Sprintf("--enable-metrics=%t", metricsEnabled),
	}

	debugEnabled := getEnableDebug(annotations)
	debugPort := getDebugPort(annotations)
	if debugEnabled {
		ports = append(ports, corev1.ContainerPort{
			Name:          sidecarDebugPortName,
			ContainerPort: int32(debugPort),
		})

		cmd = []string{"/dlv"}

		args = append([]string{
			fmt.Sprintf("--listen=:%v", debugPort),
			"--accept-multiclient",
			"--headless=true",
			"--log",
			"--api-version=2",
			"exec",
			"/rusid",
			"--",
		}, args...)
	}

	c := &corev1.Container{
		Name:            sidecarContainerName,
		Image:           rusiSidecarImage,
		ImagePullPolicy: pullPolicy,
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: &allowPrivilegeEscalation,
		},
		Ports:   ports,
		Command: cmd,
		Env: []corev1.EnvVar{
			{
				Name:  "NAMESPACE",
				Value: namespace,
			},
		},
		Args: args,
		ReadinessProbe: &corev1.Probe{
			Handler:             httpHandler,
			InitialDelaySeconds: getInt32AnnotationOrDefault(annotations, rusiReadinessProbeDelayKey, defaultHealthzProbeDelaySeconds),
			TimeoutSeconds:      getInt32AnnotationOrDefault(annotations, rusiReadinessProbeTimeoutKey, defaultHealthzProbeTimeoutSeconds),
			PeriodSeconds:       getInt32AnnotationOrDefault(annotations, rusiReadinessProbePeriodKey, defaultHealthzProbePeriodSeconds),
			FailureThreshold:    getInt32AnnotationOrDefault(annotations, rusiReadinessProbeThresholdKey, defaultHealthzProbeThreshold),
		},
		LivenessProbe: &corev1.Probe{
			Handler:             httpHandler,
			InitialDelaySeconds: getInt32AnnotationOrDefault(annotations, rusiLivenessProbeDelayKey, defaultHealthzProbeDelaySeconds),
			TimeoutSeconds:      getInt32AnnotationOrDefault(annotations, rusiLivenessProbeTimeoutKey, defaultHealthzProbeTimeoutSeconds),
			PeriodSeconds:       getInt32AnnotationOrDefault(annotations, rusiLivenessProbePeriodKey, defaultHealthzProbePeriodSeconds),
			FailureThreshold:    getInt32AnnotationOrDefault(annotations, rusiLivenessProbeThresholdKey, defaultHealthzProbeThreshold),
		},
	}
	c.Env = append(c.Env, utils.ParseEnvString(annotations[rusiEnvKey])...)
	if tokenVolumeMount != nil {
		c.VolumeMounts = []corev1.VolumeMount{
			*tokenVolumeMount,
		}
	}

	resources, err := getResourceRequirements(annotations)
	if err != nil {
		klog.Warningf("couldn't set container resource requirements: %s. using defaults", err)
	}
	if resources != nil {
		c.Resources = *resources
	}
	return c, nil
}

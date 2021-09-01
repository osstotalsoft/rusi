package modes

// RusiMode is the runtime mode for Rusi.
type RusiMode string

const (
	KubernetesMode RusiMode = "kubernetes"
	StandaloneMode RusiMode = "standalone"
)

package runtime

type Runtime struct {
	Config Config
}

func NewRuntime(config Config) *Runtime {
	return &Runtime{
		config,
	}
}

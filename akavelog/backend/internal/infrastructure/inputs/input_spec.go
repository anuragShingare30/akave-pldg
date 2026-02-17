package inputs

// InputSpec describes an input instance to be created dynamically.
// Used when mounting multiple inputs from configuration (e.g. MountHTTPEndpoints).
type InputSpec struct {
	Type        string
	Description string
	Config      Config
}

// ConfigWithDescription returns a copy of Config with description set.
func (s InputSpec) ConfigWithDescription() Config {
	cfg := make(Config, len(s.Config)+1)
	for k, v := range s.Config {
		cfg[k] = v
	}
	if s.Description != "" {
		cfg["description"] = s.Description
	}
	return cfg
}

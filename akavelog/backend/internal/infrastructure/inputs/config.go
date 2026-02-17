package inputs

// Config is a key-value map for input-type-specific configuration.
// The backend passes it when creating an input; implementations interpret it.
type Config map[string]any

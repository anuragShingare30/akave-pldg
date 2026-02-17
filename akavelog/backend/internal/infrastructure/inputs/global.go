package inputs

// GlobalRegistry is the default registry. Input implementations (e.g. httpinput) register in init().
var GlobalRegistry = NewRegistry()

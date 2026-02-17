package inputs

// ConfigField describes one configuration field for an input type.
type ConfigField struct {
	Name        string `json:"name"`
	Type        string `json:"type"`        // "string", "number", "bool", "object"
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

// InputTypeInfo describes an input type and the configuration it expects.
// Returned by Factory.ConfigSpec() and exposed via GET /inputs/info and GET /inputs/types/:type.
type InputTypeInfo struct {
	Type        string       `json:"type"`
	Description string       `json:"description"`
	Fields      []ConfigField `json:"fields"`
}

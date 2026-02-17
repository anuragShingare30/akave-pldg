package inputs

// Factory creates a MessageInput from config and buffer.
// Each input type (http, syslog, etc.) implements and registers a Factory.
// ConfigSpec declares which configuration fields this input type needs.
type Factory interface {
	Name() string
	ConfigSpec() InputTypeInfo
	Create(cfg Config, buffer InputBuffer) (MessageInput, error)
}

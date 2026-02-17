package inputs

// InputBuffer receives raw log payloads from inputs.
// The backend provides an implementation (e.g. in-memory or persistence).
type InputBuffer interface {
	Insert([]byte)
}

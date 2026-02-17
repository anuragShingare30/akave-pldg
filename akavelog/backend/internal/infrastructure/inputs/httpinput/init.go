package httpinput

import "github.com/akave-ai/akavelog/internal/infrastructure/inputs"

func init() {
	inputs.GlobalRegistry.Register(&Factory{})
}

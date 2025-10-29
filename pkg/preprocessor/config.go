package preprocessor

import (
	"os"
)

type Config struct {
	InstrumentUnsafe  bool
	InstrumentReflect bool
	Registry          *Registry
}

func LoadConfigFromEnv() (Config, error) {
	unsafe := os.Getenv("GO_INSTRUMENT_UNSAFE") == "true"
	reflect := os.Getenv("GO_INSTRUMENT_REFLECT") == "true"
	return Config{
		InstrumentUnsafe:  unsafe,
		InstrumentReflect: reflect,
		Registry:          &DefaultRegistry,
	}, nil
}

func (c Config) ShouldInstrument() bool {
	return c.InstrumentUnsafe || c.InstrumentReflect
}

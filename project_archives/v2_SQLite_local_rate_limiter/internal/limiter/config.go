package limiter

import (
	"errors"
)

type Config struct {
	Capacity        int
	TokensPerSecond float64
}

func (c Config) Validate() error {
	if c.Capacity <= 0 {
		return errors.New("capacity must be greater than 0")
	}
	if c.TokensPerSecond <= 0 {
		return errors.New("tokens per second must be greater than 0")
	}
	return nil
}

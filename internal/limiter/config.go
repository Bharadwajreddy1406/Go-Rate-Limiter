package limiter

import "fmt"

type Config struct {
	Capacity        int
	TokensPerSecond float64
}

func (c Config) Validate() error {
	if c.Capacity <= 0 {
		return fmt.Errorf("Capacity must be greater than 0")
	}
	if c.TokensPerSecond <= 0 {
		return fmt.Errorf("TokensPerSecond must be greater than 0")
	}
	return nil
}

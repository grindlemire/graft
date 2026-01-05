package shared

// Config is a shared type used by both producer and consumer
// Producer imports this package normally
// Consumer imports this package with an alias to test import alias resolution
type Config struct {
	Port int
	Host string
}

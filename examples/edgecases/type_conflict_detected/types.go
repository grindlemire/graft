package type_conflict_detected

// Shared Config type
// NOTE: This package intentionally creates a type conflict for testing purposes.
// Multiple nodes register the same output type, which should be detected as an error.
type Config struct {
	Port int
}

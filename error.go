package migrathor

// DriverError records original sql driver error and supporting info that caused it.
type DriverError struct {
	// Info contains supporting info
	Info string

	// Err is the original (possibly driver-specific) error
	Err error
}

func (e *DriverError) Error() string { return e.Info + ": " + e.Err.Error() }

// UnderlyingError returns the underlying error from DriverError.
func UnderlyingError(err error) error {
	switch err := err.(type) {
	case *DriverError:
		return err.Err
	}
	return err
}

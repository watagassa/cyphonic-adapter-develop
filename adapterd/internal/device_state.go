package internal

// deviceState represents the state of a Device.
// There are three states: down, up, closed.
// Transitions:
//
//	down -----+
//	  ↑↓      ↓
//	  up -> closed
type deviceState uint32

const (
	deviceStateDown   deviceState = iota // device has not been configured yet. or deleted (default).
	deviceStateUp                        // device is running.
	deviceStateClosed                    // device is stopping.
)

package domain

type StatusCode int

const (
	ProcessStatusCodeNoError      StatusCode = 0 // No error
	ProcessStatusCodeAbnormalExit StatusCode = 1 // Adapter daemon was forced to shut down
	ProcessStatusCodeConfigError  StatusCode = 2 // This device is not configured correctly
)

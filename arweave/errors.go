package arweave

import "errors"

// Predefined package errors.
var (
	ErrFailedToCalcPrice  = errors.New("failed to calculate arweave transaction price")
	ErrFailedToUploadData = errors.New("failed to upload data to arweave")
)

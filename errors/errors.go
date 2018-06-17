package errors

import "fmt"

// InvalidMethodError indicates that the message being
// parsed does not match the Message implementation
// this is usually user error.
type InvalidMethodError struct {
	Expected string
	Actual   string
}

func (e InvalidMethodError) Error() string {
	return fmt.Sprintf("Expected Method %s but got %s", e.Expected, e.Actual)
}

/*
UnsupportedSipVersionError indicates that some version
of the SIP protocol other than 2.0 was specified.
*/
type UnsupportedSipVersionError struct {
	Version float32
}

func (e UnsupportedSipVersionError) Error() string {
	return fmt.Sprintf("Unsupported SIP version: %f", e.Version)
}

/*
InvalidMessageFormatError indicates that a message failed to parse
due to invalid format
*/
type InvalidMessageFormatError string

func (e InvalidMessageFormatError) Error() string {
	return "Invalid Message Format: " + string(e)
}

/*
HeaderParseError indicates a problem in parsing a header
it includes the line of the header as well as the message
*/
type HeaderParseError struct {
	Message string
	Line    int
}

func (e HeaderParseError) Error() string {
	return fmt.Sprintf(
		"Error parsing header on line %d. Full Message: \n%s",
		e.Line,
		e.Message,
	)
}

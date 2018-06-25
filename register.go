package slurp

import (
	"fmt"
	"strings"
)

type Register struct {
	headers CommonHeaders
	control CallControlHeaders
	payload []byte
	uri     string
}

func (r *Register) Render() string {
	// REGISTER messages have a different URI structure, per RFC
	r.uri = r.Headers().To.Uri()
	// @ and 'user-info' components should be stripped, leaving only the domain/host
	r.uri = r.uri[strings.Index(r.uri, "@")+1:]
	return fmt.Sprintf(
		"REGISTER sip:%s SIP/2.0\r\n%s\r\n%s\r\n%s\r\n\r\n",
		r.uri,
		renderHeaders(r.headers, r.control),
		// we set CSeq outside of renderHeaders because it's method-dependent
		"CSeq: "+fmt.Sprintf("%d", r.control.Sequence)+" REGISTER",
		"Supported: SUBSCRIBE, NOTIFY",
	)
}

// Parse takes a string representation of a message and unmarshalls
// the data into the appropriate struct fields.
func (r *Register) Parse(message string) (err error) {
	// split lines
	lines := strings.Split(message, "\n")
	// ensure that the message is an Register message
	// and the the protocol is SIP/2.0
	err = validateMethod(lines[0], "REGISTER")
	// In a Register, URI should immediately follow Register
	// TODO when enough infrastructure exists to accomplish it, add support for checking for unsupported URI schemes and responding with 416
	r.uri = strings.Split(lines[0], " ")[1]
	r.headers = CommonHeaders{}
	r.control = CallControlHeaders{}
	parseHeaders(lines, &r.headers, &r.control)
	return
}

func (r *Register) Method() string {
	return "Register"
}

func (r *Register) Headers() *CommonHeaders {
	return &r.headers
}

func (r *Register) Control() *CallControlHeaders {
	return &r.control
}

func (r *Register) Payload() []byte {
	return r.payload
}

func (r *Register) StringPayload() string {
	return string(r.payload)
}

func (r *Register) SetPayload(data []byte) {
	r.payload = data
}

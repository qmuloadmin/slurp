package slurp

import (
	"fmt"
	"strings"
)

type Invite struct {
	headers CommonHeaders
	control CallControlHeaders
	raw     string
	payload []byte
	uri     string
}

func (i *Invite) Render() string {
	return fmt.Sprintf(
		"INVITE sip:%s SIP/2.0\r\n%s\r\n%s\r\n%s\r\n\r\n",
		i.Headers().To.Uri(),
		renderHeaders(i.headers, i.control),
		// we set CSeq outside of renderHeaders because it's method-dependent
		"CSeq: "+fmt.Sprintf("%d", i.control.Sequence)+" INVITE",
		"Supported: SUBSCRIBE, NOTIFY",
	)
}

// Parse takes a string representation of a message and unmarshalls
// the data into the appropriate struct fields.
func (i *Invite) Parse(message string) (err error) {
	// split lines
	lines := strings.Split(message, "\n")
	// ensure that the message is an INVITE message
	// and the the protocol is SIP/2.0
	err = validateMethod(lines[0], "INVITE")
	// In an INVITE, URI should immediate follow INVITE
	// TODO when enough infrastructure exists to accomplish it, add support for checking for unsupported URI schemes and responding with 416
	i.uri = strings.Split(lines[0], " ")[1]
	i.headers = CommonHeaders{}
	i.control = CallControlHeaders{}
	parseHeaders(lines, &i.headers, &i.control)
	return
}

func (i *Invite) Method() string {
	return "INVITE"
}

func (i *Invite) Headers() *CommonHeaders {
	return &i.headers
}

func (i *Invite) RawHeaders() string {
	return i.raw
}

func (i *Invite) Control() *CallControlHeaders {
	return &i.control
}

func (i *Invite) Payload() []byte {
	return i.payload
}

func (i *Invite) StringPayload() string {
	return string(i.payload)
}

func (i *Invite) SetPayload(data []byte) {
	i.payload = data
}

package slurp

/*
Messages are models for the marshaling and unmarshaling of data from and to raw SIP messages
*/

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	. "github.com/qmuloadmin/slurp/errors"
)

// SupportedMethods is a list of all request types currently supported by Slurp
var SupportedMethods = [5]string{
	"INVITE", "REGISTER", "NOTIFY", "SUBSCRIBE", "ACK",
}

// SupportedResponses is a mapping of support response codes and their text values
var SupportedResponses = map[int]string{
	200: "OK",
	100: "Trying",
	180: "Ringing",
	183: "Session Progress",
	401: "Unauthorized",
	404: "Not Found",
	486: "Busy Here",
}

// Message is the golang model representing an entire SIP message
type Message interface {
	Render() string
	// given a string representation of a message, unmarshall into Message object
	Parse(string) error
	// Get the method/request type of the message
	Method() string
	Headers() *CommonHeaders
	Control() *CallControlHeaders
	Payload() []byte
	StringPayload() string
	SetPayload([]byte)
}

type Invite struct {
	headers CommonHeaders
	control CallControlHeaders
	payload []byte
	uri     string
}

func (i *Invite) Render() string {
	if i.uri == "" {
		i.uri = i.headers.ToUri
	}
	return fmt.Sprintf(
		"INVITE sip:%s SIP/2.0\r\n%s\r\n%s\r\n\r\n",
		i.uri,
		renderHeaders(i.headers, i.control),
		// we set CSeq outside of renderHeaders because it's method-dependent
		"CSeq: "+fmt.Sprintf("%d", i.control.Sequence)+" INVITE",
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

// Contains header information common across all messages
type CommonHeaders struct {
	// The "common name" like "Bob" or "Sally"
	To string
	// "The URI, like 123@example.com"
	ToUri         string
	From          string
	FromUri       string
	Contact       string
	Forwards      int
	UserAgent     string
	ContentType   string
	ContentLength int
}

// CallControlHeaders are common headers that are usually only set by the system, not by users
type CallControlHeaders struct {
	// A slice of Via headers
	// the format is an array[2] of strings, where:
	// [0] = The transport (UDP, TCP)
	// [1] = The URI
	Via [][2]string
	// The branch of the most recent via, or ours if we added it
	ViaBranch    string
	CallId       string
	Sequence     int
	Authenticate string
	// Sets the To Tag element
	ToTag   string
	FromTag string
}

// Utility functions

// Make sure that the Method line of a request (the first line)
// is of the expected type for the given Message implementation
func validateMethod(line string, method string) (err error) {
	line = strings.TrimSpace(line)
	// Make sure that the request's method matches 'method'
	if !strings.HasPrefix(strings.ToUpper(line), method) {

		err = InvalidMethodError{
			Expected: method,
			Actual:   strings.Split(line, " ")[0],
		}
	}
	// Make sure version is supported. Right now only 2.0 is supported
	if !strings.HasSuffix(line, "SIP/2.0") {
		proto := strings.Split(line, " ")[2]
		version, parseErr := strconv.ParseFloat(
			strings.Split(proto, "/")[1],
			32,
		)
		if parseErr != nil {
			return InvalidMessageFormatError(line)
		}
		err = UnsupportedSipVersionError{
			Version: float32(version),
		}
	}
	return
}

func parseHeaders(lines []string, h *CommonHeaders, c *CallControlHeaders) error {
	for i, line := range lines[1:] {
		var err error
		// SplitN returns one substring per count, so 2 means "split once"
		// Go is weird sometimes
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			// if the line was only spaces, we're done with headers
			break
		}
		parts := strings.SplitN(line, ":", 2)
		_type := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		// Match each header with its name, or short form identifier
		switch strings.ToLower(_type) {
		// Note: SIP integer values must fit within 32 bit width
		case "max-forwards":
			var tempInt int64
			tempInt, err = strconv.ParseInt(value, 10, 32)
			h.Forwards = int(tempInt)
		case "contact", "m":
			h.Contact = value
		case "content-type", "c":
			h.ContentType = value
		case "content-length", "l":
			var tempInt int64
			tempInt, err = strconv.ParseInt(value, 10, 32)
			h.ContentLength = int(tempInt)
		case "via", "v":
			// strip off all parameters and store them in a slice
			// at the moment, we ignore them for reading purposes
			// (we use branch when writing)
			via := strings.SplitN(value, ";", 2)
			parts := strings.Split(via[0], " ")
			transportParts := strings.Split(parts[0], "/")
			transport := transportParts[len(transportParts)-1]
			c.Via = append(c.Via, [2]string{
				transport, parts[1],
			})
		case "cseq":
			var temp int64
			// NOTE: At the moment, we're going to assume CSeq method is valid
			parts := strings.Split(value, " ")
			// CSeq must be 32 bit
			temp, err = strconv.ParseInt(parts[0], 10, 32)
			c.Sequence = int(temp)
		case "call-id", "i":
			c.CallId = value
		case "from", "f":
			err = parseFromTo(value, &h.From, &h.FromUri, &c.FromTag)
		case "to", "t":
			err = parseFromTo(value, &h.To, &h.ToUri, &c.ToTag)
		default:
			log.Printf("Ignoring Unrecognized Header: %s", line)
		}
		if err != nil {
			message := strings.Join(lines, "")
			return HeaderParseError{
				Line:    i,
				Message: message,
			}
		}
	}
	return nil
}

func parseFromTo(value string, from *string, uri *string, tag *string) (err error) {
	// split off main header from parameters
	params := strings.Split(value, ";")
	// assign the alias/name and uri separately
	// format is NAME [space] <URI>
	// but NAME may be in "" to include a space
	// so for now, split on angle bracket, even though this isn't perfect
	parts := strings.Split(params[0], "<")
	*from = strings.TrimSpace(parts[0])
	*uri = strings.Replace(parts[1], ">", "", 1)
	// now find the from tag, if present, and store it
	for _, param := range params[1:] {
		if strings.HasPrefix(param, "tag=") {
			parts = strings.SplitN(param, "=", 2)
			*tag = parts[1]
		}
	}
	return
}

func renderHeaders(h CommonHeaders, c CallControlHeaders) string {
	lines := make([]string, 0, 10)
	// Via, From, Contact, Call-ID and CSeq must always be included

	// For sending a request, as we are a client or a server and not a Proxy
	// we only should send one Via, ourselves.
	via := fmt.Sprintf(
		// TODO need to update transport dynamically once infra is built
		"Via: SIP/2.0/%s %s;branch=%s",
		c.Via[0][0], c.Via[0][1], c.ViaBranch,
	)
	lines = append(lines, via)

	// set max forwards. RFC recommends this goes as one of first fields
	if h.Forwards == 0 {
		// Since we aren't a proxy, we're never forwarding requests. Set it to 70.
		h.Forwards = 70
	}
	forwards := fmt.Sprintf("Max-Forwards: %d", h.Forwards)
	lines = append(lines, forwards)

	from := fmt.Sprintf(
		// when rendering, there will always be a tag in From
		"From: %s <%s>;tag=%s",
		h.From, h.FromUri, c.FromTag,
	)
	lines = append(lines, from)

	// If To is set, populate To next
	to := fmt.Sprintf(
		"To: %s <%s>",
		h.To, h.ToUri,
	)
	if c.ToTag != "" {
		to += ";tag=" + c.ToTag
	}
	lines = append(lines, to)

	// Set contact always. If Contact is empty, use From
	if h.Contact == "" {
		h.Contact = h.FromUri
	}
	contact := fmt.Sprintf("Contact: <%s>", h.Contact)
	lines = append(lines, contact)

	// set call id
	id := fmt.Sprintf("Call-ID: %s", c.CallId)
	lines = append(lines, id)

	// set content type and length, if present
	if h.ContentType != "" {
		_type := fmt.Sprintf("Content-Type: %s", h.ContentType)
		length := fmt.Sprintf("Content-Length: %d", h.ContentLength)
		lines = append(lines, _type)
		lines = append(lines, length)
	}

	return strings.Join(lines, "\r\n")
}

// TODO
func generateTag() {

}

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
	RawHeaders() string
	Control() *CallControlHeaders
	Payload() []byte
	StringPayload() string
	SetPayload([]byte)
}

// Contains header information common across all messages
type CommonHeaders struct {
	To            Header
	From          Header
	Contacts      []Header
	Forward       int //MaxForwards
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

func parseParams(header string) map[string]string {
	panic("Not Implemented")
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
			h.Forward = int(tempInt)
		case "contact", "m":
			// Contact is repeatable. Each Contact can have a friendly name, URI and params
			// URI parameters are also possible but currently unsupported
			// split on comma first, which gives us multiple contacts, if present
			contacts := strings.Split(value, ",")
			for _, each := range contacts {
				contact := &Contact{}
				// split the contact on ; to find params and the value/uri
				parts := strings.Split(each, ";")
				nameAndUri := strings.Split(parts[0], "<")
				contact.SetValue(strings.TrimSpace(nameAndUri[0]))
				if len(nameAndUri) > 1 {
					uri := strings.TrimSpace(strings.Replace(nameAndUri[1], ">", "", -1))
					contact.SetUri(uri)
				}
				// Now parse each parameter
				for _, param := range parts[1:] {
					parts := strings.SplitN(param, "=", 2)
					contact.SetParam(strings.ToLower(parts[0]), parts[1])
				}
				h.Contacts = append(h.Contacts, contact)
			}
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
			if h.From == nil {
				h.From = NewHeader(&ToFrom{})
			}
			err = parseFromTo(value, h.From)
		case "to", "t":
			if h.To == nil {
				h.To = NewHeader(&ToFrom{})
			}
			err = parseFromTo(value, h.To)
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

func parseFromTo(value string, from Header) (err error) {
	// split off main header from parameters
	params := strings.Split(value, ";")
	// assign the alias/name and uri separately
	// format is NAME [space] <URI>
	// but NAME may be in "" to include a space
	// so for now, split on angle bracket, even though this isn't perfect
	parts := strings.Split(params[0], "<")
	from.SetValue(strings.TrimSpace(parts[0]))
	from.SetUri(strings.Replace(parts[1], ">", "", 1))
	// now find the from tag, if present, and store it
	for _, param := range params[1:] {
		if strings.HasPrefix(param, "tag=") {
			parts = strings.SplitN(param, "=", 2)
			from.SetParam("tag", parts[1])
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
	if h.Forward == 0 {
		// Since we aren't a proxy, we're never forwarding requests. Set it to 70.
		h.Forward = 70
	}
	forwards := fmt.Sprintf("Max-Forwards: %d", h.Forward)
	lines = append(lines, forwards)

	from := fmt.Sprintf(
		// when rendering, there will always be a tag in From
		"From: %s <%s>;tag=%s",
		h.From.Value(), h.From.Uri(), h.From.Param("tag"),
	)
	lines = append(lines, from)

	// If To is set, populate To next
	to := fmt.Sprintf(
		"To: %s <%s>",
		h.To.Value(), h.To.Uri(),
	)
	if h.To.Param("tag") != "" {
		to += ";tag=" + h.To.Param("tag")
	}
	lines = append(lines, to)

	// Set contact always. If Contact is empty, use From
	if len(h.Contacts) == 0 {
		contact := NewHeader(&Contact{}).SetUri(h.From.Uri()).SetValue(h.From.Value())
		h.Contacts = []Header{contact}
	}
	for _, contact := range h.Contacts {
		result := "Contact: " + strings.Join([]string{contact.Value(),
			fmt.Sprintf("<%s>", contact.Uri())},
			" ")
		result += contact.ParamString()
		lines = append(lines, result)
	}

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

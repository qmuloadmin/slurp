package slurp

import (
	"fmt"
	"strings"
)

// Header represents "complicated" headers in the SIP RFC
// Not all headers are Headers. For instance, MaxForwards is
// fundamentally too simple to merit so much overhead
type Header interface {
	Value() string
	Param(string) string
	Uri() string
	SetValue(string) Header
	SetParam(string, string) Header
	SetUri(string) Header
	ParamString() string
	// Do we want to do Init() here or move Render() to each header impl?
	Init() Header
}

func NewHeader(h Header) Header {
	return h.Init()
}

// The Contact header, which is repeatable
type Contact map[string]string

func (h *Contact) Init() Header {
	*h = make(map[string]string)
	return h
}

func (h *Contact) Value() string {
	return (*h)["_value"]
}

func (h *Contact) Param(name string) string {
	return (*h)[name]
}

func (h *Contact) SetValue(value string) Header {
	(*h)["_value"] = value
	return h
}

func (h *Contact) SetParam(name, value string) Header {
	(*h)[name] = value
	return h
}

func (h *Contact) Uri() string {
	return (*h)["_uri"]
}

func (h *Contact) SetUri(uri string) Header {
	(*h)["_uri"] = uri
	return h
}

func (h *Contact) ParamString() (result string) {
	for k, v := range *h {
		if !strings.HasPrefix(k, "_") {
			result += fmt.Sprintf(
				"; %s=%s",
				k,
				v,
			)
		}
	}
	return
}

// Used for both From an To headers as they have the same parameters
type ToFrom struct {
	value string
	uri   string
	tag   string
}

func (t *ToFrom) Init() Header {
	return t
}

func (t *ToFrom) Value() string {
	return t.value
}

func (t *ToFrom) Param(name string) string {
	if name != "tag" {
		return ""
	}
	return t.tag
}

func (t *ToFrom) Uri() string {
	return t.uri
}

func (t *ToFrom) SetValue(value string) Header {
	t.value = value
	return t
}

func (t *ToFrom) SetParam(name, value string) Header {
	if name == "tag" {
		t.tag = value
	} // discard everything else. To shouldn't contain any other parameters
	return t
}

func (t *ToFrom) SetUri(uri string) Header {
	t.uri = uri
	return t
}

func (t *ToFrom) ParamString() string {
	return "; tag=" + t.tag
}

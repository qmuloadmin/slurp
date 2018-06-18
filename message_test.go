package slurp

import (
	"fmt"
	"io/ioutil"
	"strings"
	"testing"

	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
)

func TestParseInvite(t *testing.T) {
	if data, err := ioutil.ReadFile("examples/invite.sip"); err == nil {
		text := string(data)
		message := Invite{}
		err = message.Parse(text)
		if err != nil {
			t.Fail()
		}
		// check to ensure fields unmarshalled correctly
		assert.Equal(t, message.Method(), "INVITE")
		assert.Equal(t, message.Headers().To, "Bob")
		assert.Equal(t, message.Headers().ContentType, "application/sdp")
		assert.Equal(t, message.Headers().ToUri, "sip:bob@biloxi.com")
		assert.Equal(t, message.Headers().From, "Alice")
		assert.Equal(t, message.Headers().FromUri, "sip:alice@atlanta.com")
		assert.Equal(t, message.Control().Sequence, 314159)
		assert.Equal(t, message.Control().FromTag, "1928301774")
		assert.Equal(t, message.Control().CallId, "a84b4c76e66710@pc33.atlanta.com")
		assert.Equal(t, message.Control().Via[0][0], "UDP")
		assert.Equal(t, message.Control().Via[0][1], "pc33.atlanta.com")
	}
}

func TestRenderInvite(t *testing.T) {
	callId := uuid.New()
	expected := fmt.Sprintf(`INVITE sip:sally@nasa.gov SIP/2.0
Via: SIP/2.0/TCP 192.168.1.2;branch=z9hG4bKg56fd
Max-Forwards: 70
From: Geoff <gharding@test.com>;tag=5gh941c
To: Sally <sally@nasa.gov>
Contact: <gharding@test.com>
Call-ID: %s
CSeq: 4 INVITE

`, callId.String())
	expected = strings.Replace(expected, "\n", "\r\n", -1)
	invite := Invite{}
	headers := invite.Headers()
	control := invite.Control()
	headers.To = "Sally"
	headers.ToUri = "sally@nasa.gov"
	headers.From = "Geoff"
	headers.FromUri = "gharding@test.com"
	control.FromTag = "5gh941c"
	control.CallId = callId.String()
	control.Sequence = 4
	control.Via = [][2]string{[2]string{"TCP", "192.168.1.2"}}
	control.ViaBranch = "z9hG4bKg56fd"
	headers.Contact = "gharding@test.com"
	headers.UserAgent = "slurp"
	rendered := invite.Render()
	t.Log("Rendered Invite: " + rendered)
	assert.Equal(t, expected, rendered)
}

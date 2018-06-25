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
		assert.Equal(t, message.Headers().To.Value(), "Bob")
		assert.Equal(t, message.Headers().ContentType, "application/sdp")
		assert.Equal(t, message.Headers().To.Uri(), "sip:bob@biloxi.com")
		assert.Equal(t, message.Headers().From.Value(), "Alice")
		assert.Equal(t, message.Headers().From.Uri(), "sip:alice@atlanta.com")
		assert.Equal(t, message.Control().Sequence, 314159)
		assert.Equal(t, message.Headers().From.Param("tag"), "1928301774")
		assert.Equal(t, message.Control().CallId, "a84b4c76e66710@pc33.atlanta.com")
		assert.Equal(t, message.Control().Via[0][0], "UDP")
		assert.Equal(t, message.Control().Via[0][1], "pc33.atlanta.com")
	}
}

func TestParseRegister(t *testing.T) {
	if data, err := ioutil.ReadFile("examples/register.sip"); err == nil {
		text := string(data)
		message := Register{}
		err = message.Parse(text)
		if err != nil {
			t.Fail()
		}
		// check to ensure fields unmarshalled correctly
		assert.Equal(t, message.Method(), "REGISTER")
		assert.Equal(t, message.Headers().To.Value(), "Bob")
		assert.Equal(t, message.Headers().To.Uri(), "sip:bob@biloxi.com")
		assert.Equal(t, message.Headers().From.Value(), "Bob")
		assert.Equal(t, message.Headers().From.Uri(), "sip:bob@biloxi.com")
		assert.Equal(t, message.Control().Sequence, 314)
		assert.Equal(t, "54548", message.Headers().From.Param("tag"))
		assert.Equal(t, message.Control().CallId, "a84b4c76e66710@pc33.atlanta.com")
		assert.Equal(t, message.Control().Via[0][0], "TCP")
		assert.Equal(t, message.Control().Via[0][1], "pc33.atlanta.com")
		assert.Equal(t, "biloxi.com", message.Uri())
	}
}

func TestRenderInvite(t *testing.T) {
	callId := uuid.New()
	expected := fmt.Sprintf(`INVITE sip:sally@nasa.gov SIP/2.0
Via: SIP/2.0/TCP 192.168.1.2;branch=z9hG4bKg56fd
Max-Forwards: 70
From: Geoff <gharding@test.com>;tag=5gh941c
To: Sally <sally@nasa.gov>
Contact: Geoff <gharding@test.com>
Call-ID: %s
CSeq: 4 INVITE
Supported: SUBSCRIBE, NOTIFY

`, callId.String())
	expected = strings.Replace(expected, "\n", "\r\n", -1)
	invite := Invite{}
	headers := invite.Headers()
	control := invite.Control()
	headers.To = NewHeader(&ToFrom{}).SetValue("Sally").SetUri("sally@nasa.gov")
	headers.From = NewHeader(&ToFrom{}).SetValue("Geoff").SetUri("gharding@test.com").SetParam("tag", "5gh941c")
	control.CallId = callId.String()
	control.Sequence = 4
	control.Via = [][2]string{[2]string{"TCP", "192.168.1.2"}}
	control.ViaBranch = "z9hG4bKg56fd"
	headers.UserAgent = "slurp"
	rendered := invite.Render()
	t.Log("Rendered Invite: " + rendered)
	assert.Equal(t, expected, rendered)
}

func TestRenderRegister(t *testing.T) {
	callId := uuid.New()
	expected := fmt.Sprintf(`REGISTER sip:nasa.gov SIP/2.0
Via: SIP/2.0/TCP 192.168.1.2;branch=z9hG4bKg56fd
Max-Forwards: 70
From: Sally <sally@nasa.gov>;tag=5gh941c
To: Sally <sally@nasa.gov>
Contact: Sally <sally@nasa.gov>
Call-ID: %s
CSeq: 4 REGISTER
Supported: SUBSCRIBE, NOTIFY

`, callId.String())
	expected = strings.Replace(expected, "\n", "\r\n", -1)
	register := Register{}
	headers := register.Headers()
	control := register.Control()
	headers.To = NewHeader(&ToFrom{}).SetUri("sally@nasa.gov").SetValue("Sally")
	headers.From = NewHeader(&ToFrom{}).SetValue("Sally").SetUri("sally@nasa.gov").SetParam("tag", "5gh941c")
	control.CallId = callId.String()
	control.Sequence = 4
	control.Via = [][2]string{[2]string{"TCP", "192.168.1.2"}}
	control.ViaBranch = "z9hG4bKg56fd"
	headers.UserAgent = "slurp"
	rendered := register.Render()
	t.Log("Rendered Register: " + rendered)
	assert.Equal(t, expected, rendered)
}

package slurp

import (
	"io/ioutil"
	"testing"

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

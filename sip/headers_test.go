package sip

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrependHeader(t *testing.T) {
	hs := headers{}

	hs.PrependHeader(&ViaHeader{})
	assert.Equal(t, 1, len(hs.headerOrder))

	v := &ViaHeader{}
	hs.PrependHeader(v.Clone())
	assert.Equal(t, 2, len(hs.headerOrder))
	assert.Equal(t, v, hs.GetHeader("via"))
}

func BenchmarkHeadersPrepend(b *testing.B) {
	callID := CallIDHeader("aaaa")
	hs := headers{
		headerOrder: []Header{
			&ViaHeader{},
			&FromHeader{},
			&ToHeader{},
			&CSeqHeader{},
			&callID,
			&ContactHeader{},
		},
	}

	var header Header = &ViaHeader{}

	b.Run("Append", func(b *testing.B) {
		for b.Loop() {
			newOrder := make([]Header, 1, len(hs.headerOrder)+1)
			newOrder[0] = header
			hs.headerOrder = append(newOrder, hs.headerOrder...)
		}
	})

	// Our version must be faster than GOSIP
	b.Run("Assign", func(b *testing.B) {
		for b.Loop() {
			newOrder := make([]Header, len(hs.headerOrder)+1)
			newOrder[0] = header
			for i, h := range hs.headerOrder {
				newOrder[i+1] = h
			}
			hs.headerOrder = newOrder
		}
	})
}

func TestLazyParsing(t *testing.T) {
	headers := new(headers)

	t.Run("Contact", func(t *testing.T) {
		headers.AppendHeader(NewHeader("Contact", "<sip:alice@example.com>"))
		h := headers.Contact()
		require.NotNil(t, h)
		require.Equal(t, "<sip:alice@example.com>", h.Value())
	})

	t.Run("Via", func(t *testing.T) {
		headers.AppendHeader(NewHeader("Via", "SIP/2.0/UDP 10.1.1.1:5060;branch=z9hG4bKabcdef"))
		h := headers.Via()
		require.NotNil(t, h)
		require.Equal(t, "SIP/2.0/UDP 10.1.1.1:5060;branch=z9hG4bKabcdef", h.Value())
	})

}

func BenchmarkLazyParsing(b *testing.B) {
	headers := new(headers)
	headers.AppendHeader(NewHeader("Contact", "<sip:alice@example.com>"))

	for b.Loop() {
		c := headers.Contact()
		if c == nil {
			b.Fatal("contact is nil")
		}
		headers.contact = nil
	}
}

func TestMaxForwardIncDec(t *testing.T) {
	maxfwd := MaxForwardsHeader(70)
	maxfwd.Dec()
	assert.Equal(t, uint32(69), maxfwd.Val(), "Value returned %d", maxfwd.Val())
}

func TestCopyHeaders(t *testing.T) {
	invite, _, _ := testCreateInvite(t, "sip:bob@example.com", "udp", "test.com")
	invite.AppendHeader(NewHeader("Record-Route", "<sip:p1:5060;lr;transport=udp>"))
	invite.AppendHeader(NewHeader("Record-Route", "<sip:p2:5060;lr>"))

	res := NewResponse(StatusOK, "OK")
	CopyHeaders("Record-Route", invite, res)

	hdrs := res.GetHeaders("Record-Route")
	require.Equal(t, "Record-Route: <sip:p1:5060;lr;transport=udp>", hdrs[0].String())
	require.Equal(t, "Record-Route: <sip:p2:5060;lr>", hdrs[1].String())
}

func TestHeaderClone(t *testing.T) {
	via := &ViaHeader{
		ProtocolName:    "SIP",
		ProtocolVersion: "2.0",
		Host:            "test.com",
		Port:            5060,
		Params:          NewParams().Add(Pair{"branch", "z9hG4bKabcdef"}),
	}
	clone := via.Clone()
	assert.Equal(t, via.ProtocolName, clone.ProtocolName)
	assert.Equal(t, via.ProtocolVersion, clone.ProtocolVersion)
	assert.Equal(t, via.Host, clone.Host)
	assert.Equal(t, via.Port, clone.Port)
	assert.Equal(t, via.Params, clone.Params)
	assert.NotSame(t, via, clone, "Clone should not be the same instance")
}

func TestHeaders_String(t *testing.T) {
	t.Run("Empty headers", func(t *testing.T) {
		hs := &headers{}
		assert.Equal(t, "\r\n", hs.String())
	})

	t.Run("Single header", func(t *testing.T) {
		hs := &headers{}
		hs.AppendHeader(NewHeader("X-Test", "abc"))
		assert.Equal(t, "X-Test: abc\r\n", hs.String())
	})

	t.Run("Multiple headers", func(t *testing.T) {
		hs := &headers{}
		hs.AppendHeader(NewHeader("X-First", "1"))
		hs.AppendHeader(NewHeader("X-Second", "2"))
		hs.AppendHeader(NewHeader("X-Third", "3"))
		expected := "X-First: 1\r\nX-Second: 2\r\nX-Third: 3\r\n"
		assert.Equal(t, expected, hs.String())
	})

	t.Run("Headers with special values", func(t *testing.T) {
		hs := &headers{}
		hs.AppendHeader(NewHeader("X-Empty", ""))
		hs.AppendHeader(NewHeader("X-Colon", "a:b"))
		expected := "X-Empty: \r\nX-Colon: a:b\r\n"
		assert.Equal(t, expected, hs.String())
	})

	t.Run("Headers with known types", func(t *testing.T) {
		hs := &headers{}
		via := &ViaHeader{
			ProtocolName:    "SIP",
			ProtocolVersion: "2.0",
			Transport:       "UDP",
			Host:            "host.com",
			Port:            5060,
			Params:          NewParams().Add(Pair{"branch", "z9hG4bK"}),
		}
		callID := CallIDHeader("callid-123")
		hs.AppendHeader(via)
		hs.AppendHeader(&callID)
		expected := "Via: SIP/2.0/UDP host.com:5060;branch=z9hG4bK\r\nCall-ID: callid-123\r\n"
		assert.Equal(t, expected, hs.String())
	})
}

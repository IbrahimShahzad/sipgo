package sip

import (
	"fmt"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testCreateMessage(t testing.TB, rawMsg []string) Message {
	msg, err := ParseMessage([]byte(strings.Join(rawMsg, "\r\n")))
	if err != nil {
		t.Fatal(err)
	}
	return msg
}

func testCreateRequest(t testing.TB, method string, targetSipUri string, transport, fromAddr string) (r *Request) {
	branch := GenerateBranch()
	callid := "gotest-" + time.Now().Format(time.RFC3339Nano)
	ftag := fmt.Sprintf("%d", time.Now().UnixNano())
	return testCreateMessage(t, []string{
		method + " " + targetSipUri + " SIP/2.0",
		"Via: SIP/2.0/" + transport + " " + fromAddr + ";branch=" + branch,
		"From: \"Alice\" <sip:alice@" + fromAddr + ">;tag=" + ftag,
		"To: \"Bob\" <" + targetSipUri + ">",
		"Call-ID: " + callid,
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}).(*Request)
}

func testCreateInvite(t testing.TB, targetSipUri string, transport, fromAddr string) (r *Request, callid string, ftag string) {
	branch := GenerateBranch()
	callid = "gotest-" + time.Now().Format(time.RFC3339Nano)
	ftag = fmt.Sprintf("%d", time.Now().UnixNano())
	return testCreateMessage(t, []string{
		"INVITE " + targetSipUri + " SIP/2.0",
		"Via: SIP/2.0/" + transport + " " + fromAddr + ";branch=" + branch,
		"From: \"Alice\" <sip:alice@" + fromAddr + ">;tag=" + ftag,
		"To: \"Bob\" <" + targetSipUri + ">",
		"Call-ID: " + callid,
		"CSeq: 1 INVITE",
		"Content-Length: 0",
		"",
		"",
	}).(*Request), callid, ftag
}

func TestResolveInterfaceIP(t *testing.T) {
	ip, iface, err := ResolveInterfacesIP("ip4", nil)
	require.NoError(t, err)
	require.NotNil(t, ip)

	t.Log(ip.String(), len(ip), iface.Name)
	assert.False(t, ip.IsLoopback())
	assert.NotNil(t, ip.To4())

	ip, iface, err = ResolveInterfacesIP("ip6", nil)
	require.NoError(t, err)
	require.NotNil(t, ip)

	t.Log(ip.String(), len(ip), iface.Name)
	assert.False(t, ip.IsLoopback())
	assert.NotNil(t, ip.To16())

	ipnet := net.IPNet{
		IP:   net.ParseIP("127.0.0.1"),
		Mask: net.IPv4Mask(255, 255, 255, 0),
	}
	ip, iface, err = ResolveInterfacesIP("ip4", &ipnet)
	require.NoError(t, err)
	require.NotNil(t, ip)
}

func TestASCIIToLower(t *testing.T) {
	val := ASCIIToLower("CSeq")
	assert.Equal(t, "cseq", val)
}

func TestASCIIToLowerInPlace(t *testing.T) {
	tests := []struct {
		name     string
		input    []byte
		expected []byte
	}{
		{
			name:     "All uppercase",
			input:    []byte("ABCDEF"),
			expected: []byte("abcdef"),
		},
		{
			name:     "Mixed case",
			input:    []byte("AbCdEf"),
			expected: []byte("abcdef"),
		},
		{
			name:     "All lowercase",
			input:    []byte("abcdef"),
			expected: []byte("abcdef"),
		},
		{
			name:     "With numbers and symbols",
			input:    []byte("A1B2C3!@#"),
			expected: []byte("a1b2c3!@#"),
		},
		{
			name:     "Empty slice",
			input:    []byte(""),
			expected: []byte(""),
		},
		{
			name:     "Non-ASCII",
			input:    []byte("ÀÁÂÃÄÅ"),
			expected: []byte("ÀÁÂÃÄÅ"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			in := make([]byte, len(tc.input))
			copy(in, tc.input)
			ASCIIToLowerInPlace(in)
			assert.Equal(t, tc.expected, in)
		})
	}
}
func TestASCIIToUpper(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "All lowercase",
			input:    "abcdef",
			expected: "ABCDEF",
		},
		{
			name:     "Mixed case",
			input:    "AbCdEf",
			expected: "ABCDEF",
		},
		{
			name:     "All uppercase",
			input:    "ABCDEF",
			expected: "ABCDEF",
		},
		{
			name:     "With numbers and symbols",
			input:    "a1b2c3!@#",
			expected: "A1B2C3!@#",
		},
		{
			name:     "Empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "Non-ASCII",
			input:    "àáâãäå",
			expected: "àáâãäå",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			out := ASCIIToUpper(tc.input)
			assert.Equal(t, tc.expected, out)
		})
	}
}
func BenchmarkHeaderToLower(b *testing.B) {
	//BenchmarkHeaderToLower-8   	1000000000	         1.033 ns/op	       0 B/op	       0 allocs/op
	h := "Content-Type"
	for i := 0; i < b.N; i++ {
		s := HeaderToLower(h)
		if s != "content-type" {
			b.Fatal("Header not lowered")
		}
	}
}

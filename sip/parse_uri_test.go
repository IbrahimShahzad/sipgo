package sip

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseSIPURI(t *testing.T) {
	// This are all good accepted URIs test.

	/*
		https://datatracker.ietf.org/doc/html/rfc3261#section-19.1.3
		sip:alice@atlanta.com
		sip:alice:secretword@atlanta.com;transport=tcp
		sips:alice@atlanta.com?subject=project%20x&priority=urgent
		sip:+1-212-555-1212:1234@gateway.com;user=phone
		sips:1212@gateway.com
		sip:alice@192.0.2.4
		sip:atlanta.com;method=REGISTER?to=alice%40atlanta.com
		sip:alice;day=tuesday@atlanta.com
	*/

	var uri SIPURI
	var err error
	var str string

	t.Run("basic", func(t *testing.T) {
		uri = SIPURI{}
		str = "sip:alice@localhost:5060"
		err = ParseSIPURI(str, &uri)
		require.NoError(t, err)
		assert.Equal(t, "alice", uri.User)
		assert.Equal(t, "localhost", uri.Host)
		assert.Equal(t, 5060, uri.Port)
		assert.Equal(t, "localhost:5060", uri.HostPort())
		assert.Equal(t, "alice@localhost:5060", uri.Endpoint())
	})

	t.Run("sip case insensitive", func(t *testing.T) {
		testCases := []string{
			"sip:alice@atlanta.com",
			"SIP:alice@atlanta.com",
			"sIp:alice@atlanta.com",
		}
		for _, testCase := range testCases {
			err = ParseSIPURI(testCase, &uri)
			require.NoError(t, err)
			assert.Equal(t, "alice", uri.User)
			assert.Equal(t, "atlanta.com", uri.Host)
			assert.False(t, uri.IsEncrypted())
		}

		testCases = []string{
			"sips:alice@atlanta.com",
			"SIPS:alice@atlanta.com",
			"sIpS:alice@atlanta.com",
		}
		for _, testCase := range testCases {
			err = ParseSIPURI(testCase, &uri)
			require.NoError(t, err)
			assert.Equal(t, "alice", uri.User)
			assert.Equal(t, "atlanta.com", uri.Host)
			assert.True(t, uri.IsEncrypted())
		}

	})

	t.Run("with sip scheme slashes", func(t *testing.T) {
		// No scheme we currently allow
		uri = SIPURI{}
		str = "sip://alice@localhost:5060"
		err = ParseSIPURI(str, &uri)
		require.NoError(t, err)
		assert.Equal(t, "sip://alice@localhost:5060", uri.String())
	})

	t.Run("no sip scheme", func(t *testing.T) {
		uri = SIPURI{}
		str = "alice@localhost:5060"
		err = ParseSIPURI(str, &uri)
		require.Error(t, err)
	})

	t.Run("uri params parsed", func(t *testing.T) {
		uri = SIPURI{}
		str = "sips:alice@atlanta.com?subject=project%20x&priority=urgent"
		err = ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "alice", uri.User)
		assert.Equal(t, "atlanta.com", uri.Host)
		subject, _ := uri.Headers.Get("subject")
		priority, _ := uri.Headers.Get("priority")
		assert.Equal(t, "project%20x", subject)
		assert.Equal(t, "urgent", priority)
	})

	t.Run("header params parsed", func(t *testing.T) {
		uri = SIPURI{}
		str = "sip:bob:secret@atlanta.com:9999;rport;transport=tcp;method=REGISTER?to=sip:bob%40biloxi.com"
		err = ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "bob", uri.User)
		assert.Equal(t, "secret", uri.Password)
		assert.Equal(t, "atlanta.com", uri.Host)
		assert.Equal(t, 9999, uri.Port)

		assert.Equal(t, 3, uri.Params.Length())
		transport, _ := uri.Params.Get("transport")
		method, _ := uri.Params.Get("method")
		assert.Equal(t, "tcp", transport)
		assert.Equal(t, "REGISTER", method)

		assert.Equal(t, 1, uri.Headers.Length())
		to, _ := uri.Headers.Get("to")
		assert.Equal(t, "sip:bob%40biloxi.com", to)

	})

	t.Run("params no value", func(t *testing.T) {
		uri = SIPURI{}
		str = "sip:127.0.0.2:5060;rport;branch=z9hG4bKPj6c65c5d9-b6d0-4a30-9383-1f9b42f97de9"
		err = ParseSIPURI(str, &uri)
		require.NoError(t, err)

		rport, _ := uri.Params.Get("rport")
		branch, _ := uri.Params.Get("branch")
		assert.Equal(t, "", rport)
		assert.Equal(t, "z9hG4bKPj6c65c5d9-b6d0-4a30-9383-1f9b42f97de9", branch)
	})

}

func TestParseUriBad(t *testing.T) {
	t.Run("double ports", func(t *testing.T) {
		str := "sip:127.0.0.1:5060:5060;lr;transport=udp"
		uri := SIPURI{}
		err := ParseSIPURI(str, &uri)
		require.Error(t, err)
	})
}

func TestParseUriIPV6(t *testing.T) {
	t.Run("partial", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:[fe80::dc45:996b:6de9:9746"
		err := ParseSIPURI(str, &uri)
		require.Error(t, err)
	})

	t.Run("too long", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:[fe80::dc45:996b:6de9:9746:ffff:ffff:ffff:ffff]"
		err := ParseSIPURI(str, &uri)
		require.Error(t, err)
	})

	t.Run("smallest", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:[fe80::dc45:996b:6de9:9746]"
		err := ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "[fe80::dc45:996b:6de9:9746]", uri.Host)
		assert.Equal(t, 0, uri.Port)
		assert.Equal(t, "", uri.User)
	})
	t.Run("with port", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:[fe80::dc45:996b:6de9:9746]:5060"
		err := ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "[fe80::dc45:996b:6de9:9746]", uri.Host)
		assert.Equal(t, 5060, uri.Port)
	})

	t.Run("max length", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:5060"
		err := ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]", uri.Host)
		assert.Equal(t, 5060, uri.Port)
	})

	t.Run("with params", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:[fe80::dc45:996b:6de9:9746]:5060;rport;branch=z9hG4bKPj6c65c5d9-b6d0-4a30-9383-1f9b42f97de9"
		err := ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "[fe80::dc45:996b:6de9:9746]", uri.Host)
		assert.Equal(t, 5060, uri.Port)

		rport, _ := uri.Params.Get("rport")
		branch, _ := uri.Params.Get("branch")
		assert.Equal(t, "", rport)
		assert.Equal(t, "z9hG4bKPj6c65c5d9-b6d0-4a30-9383-1f9b42f97de9", branch)
	})

	t.Run("with params", func(t *testing.T) {
		uri := SIPURI{}
		str := "sip:user@[fe80::dc45:996b:6de9:9746]:5060;rport;branch=z9hG4bKPj6c65c5d9-b6d0-4a30-9383-1f9b42f97de9"
		err := ParseSIPURI(str, &uri)
		require.NoError(t, err)

		assert.Equal(t, "[fe80::dc45:996b:6de9:9746]", uri.Host)
		assert.Equal(t, 5060, uri.Port)
		assert.Equal(t, "user", uri.User)
	})
}

func TestParseTELURI(t *testing.T) {
	// https://datatracker.ietf.org/doc/html/rfc3966#section-6
	// tel:+1-201-555-0123
	// tel:7042;phone-context=example.com
	// tel:863-1234;phone-context=+1-914-555
	// tel:+12-(34)-56-78;Ext=200;ISUB=+123-456

	t.Run("basic tel uri", func(t *testing.T) {
		uri := &TELURI{}
		str := "tel:+1-201-555-0123"
		err := ParseTELURI(str, uri)
		require.NoError(t, err)
		assert.Equal(t, "+1-201-555-0123", uri.Number)
		assert.Equal(t, URISchemeTEL, uri.GetScheme())
	})

	t.Run("tel uri with context", func(t *testing.T) {
		uri := &TELURI{}
		str := "tel:7042;phone-context=example.com"
		err := ParseTELURI(str, uri)
		require.NoError(t, err)
		assert.Equal(t, "7042", uri.Number)
		pCtx, ok := uri.Params.Get("phone-context")
		require.True(t, ok)
		assert.Equal(t, "example.com", pCtx)
	})

	t.Run("tel uri with prefix", func(t *testing.T) {
		uri := &TELURI{}
		str := "tel:863-1234;phone-context=+1-914-555"
		err := ParseTELURI(str, uri)
		require.NoError(t, err)
		assert.Equal(t, "863-1234", uri.Number)
		pCtx, ok := uri.Params.Get("phone-context")
		require.True(t, ok)
		assert.Equal(t, "+1-914-555", pCtx)
	})

	t.Run("tel uri params", func(t *testing.T) {
		uri := &TELURI{}
		str := "tel:+12-(34)-56-78;Ext=200;ISUB=+123-456"
		err := ParseTELURI(str, uri)
		require.NoError(t, err)
		assert.Equal(t, "+12-(34)-56-78", uri.Number)
		ext, ok := uri.Params.Get("Ext")
		require.True(t, ok)
		assert.Equal(t, "200", ext)
		isub, ok := uri.Params.Get("ISUB")
		require.True(t, ok)
		assert.Equal(t, "+123-456", isub)
	})
}

func TestTEL2SIPURI(t *testing.T) {
	t.Run("error tel uri to sip without phone-contex", func(t *testing.T) {
		tel := &TELURI{}
		str := "tel:+1-201-555-0123"
		err := ParseTELURI(str, tel)
		require.NoError(t, err)

		sip := &SIPURI{}
		err = tel.TELtoSIP(sip)
		require.Error(t, err)
	})

	t.Run("tel2sip with context to sip", func(t *testing.T) {
		tel := &TELURI{}
		str := "tel:+1-201-555-0123;phone-context=example.com"
		err := ParseTELURI(str, tel)
		require.NoError(t, err)

		sip := &SIPURI{}
		err = tel.TELtoSIP(sip)
		require.NoError(t, err)
		assert.Equal(t, "sip:+12015550123@example.com;user-context=phone", sip.String())
	})

	t.Run("tel2sip with params", func(t *testing.T) {
		uri := &TELURI{}
		str := "tel:+12-(34)-56-78;Ext=200;ISUB=+123-456;phone-context=example.com"
		err := ParseTELURI(str, uri)
		require.NoError(t, err)
		sip := &SIPURI{}
		err = uri.TELtoSIP(sip)
		require.NoError(t, err)
		assert.Equal(t, "sip:+12345678@example.com;Ext=200;ISUB=+123-456;user-context=phone", sip.String())
	})

}

func TestParseUri(t *testing.T) {
	// "sip:alice@atlanta.com",
	// "SIP:alice@atlanta.com",
	// "sIp:alice@atlanta.com
	// sip:bob:secret@atlanta.com:9999;rport;transport=tcp;method=REGISTER?to=sip:bob%40biloxi.com
	// tel:863-1234;phone-context=+1-914-555
	tests := []struct {
		input string
		want  URI
		err   error
	}{
		{
			input: "sip:alice@atlanta.com",
			want: &SIPURI{
				User: "alice",
				Host: "atlanta.com",
			},
			err: nil,
		},
	}

	for _, tc := range tests {
		uri, err := ParseURI(tc.input)
		if err != tc.err {
			t.Errorf("err wanted %v, got %v", tc.err, err)
			continue
		}
		// assert.EqualError(t, err, tc.err.Error())
		assert.Equal(t, tc.input, uri.String())
		got, ok := uri.(*SIPURI)
		if !ok {
			t.Errorf("unexepected, want %v got %v", tc.want, got)
			continue
		}
		want, _ := tc.want.(*SIPURI)
		assert.Equal(t, want.User, got.User)
		assert.Equal(t, want.Host, got.Host)

	}

}

func TestParseURI_IPv6(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expected    *SIPURI
		expectError bool
	}{
		{
			name:  "IPv6 basic",
			input: "sip:[2001:db8::1]",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:db8::1]",
			},
		},
		{
			name:  "IPv6 with port",
			input: "sip:[2001:db8::1]:5060",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:db8::1]",
				Port:   5060,
			},
		},
		{
			name:  "IPv6 with user",
			input: "sip:alice@[2001:db8::1]",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				User:   "alice",
				Host:   "[2001:db8::1]",
			},
		},
		{
			name:  "IPv6 with user and password",
			input: "sip:alice:secret@[2001:db8::1]",
			expected: &SIPURI{
				Scheme:   URISchemeSIP,
				User:     "alice",
				Password: "secret",
				Host:     "[2001:db8::1]",
			},
		},
		{
			name:  "IPv6 with user, password, and port",
			input: "sip:alice:secret@[2001:db8::1]:5060",
			expected: &SIPURI{
				Scheme:   URISchemeSIP,
				User:     "alice",
				Password: "secret",
				Host:     "[2001:db8::1]",
				Port:     5060,
			},
		},
		{
			name:  "IPv6 with URI parameters",
			input: "sip:[2001:db8::1];transport=tcp",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:db8::1]",
				// Params should contain transport=tcp
			},
		},
		{
			name:  "IPv6 with port and URI parameters",
			input: "sip:[2001:db8::1]:5060;transport=tcp;lr",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:db8::1]",
				Port:   5060,
				// Params should contain transport=tcp and lr
			},
		},
		{
			name:  "IPv6 with headers",
			input: "sip:[2001:db8::1]?subject=test",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:db8::1]",
				// Headers should contain subject=test
			},
		},
		{
			name:  "IPv6 with port, params, and headers",
			input: "sip:[2001:db8::1]:5060;transport=tcp?subject=test&priority=urgent",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:db8::1]",
				Port:   5060,
			},
		},
		{
			name:  "IPv6 full address",
			input: "sip:[2001:0db8:85a3:0000:0000:8a2e:0370:7334]:8080",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[2001:0db8:85a3:0000:0000:8a2e:0370:7334]",
				Port:   8080,
			},
		},
		{
			name:  "IPv6 localhost",
			input: "sip:[::1]",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[::1]",
			},
		},
		{
			name:  "IPv6 with zone ID (if supported)",
			input: "sip:[fe80::1%eth0]",
			expected: &SIPURI{
				Scheme: URISchemeSIP,
				Host:   "[fe80::1%eth0]",
			},
		},
		{
			name:  "SIPS scheme with IPv6",
			input: "sips:[2001:db8::1]:5061",
			expected: &SIPURI{
				Scheme: URISchemeSIPS,
				Host:   "[2001:db8::1]",
				Port:   5061,
			},
		},
		// Error cases
		{
			name:        "IPv6 missing closing bracket",
			input:       "sip:[2001:db8::1",
			expectError: true,
		},
		{
			name:        "IPv6 missing opening bracket",
			input:       "sip:2001:db8::1]",
			expectError: true,
		},
		{
			name:        "IPv6 empty brackets",
			input:       "sip:[]",
			expectError: true,
		},
		// We dont validate the address content ?
		// {
		// 	name:        "IPv6 invalid characters in brackets",
		// 	input:       "sip:[invalid::address]",
		// 	expectError: true,
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := ParseURI(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, uri)
				return
			}

			require.NoError(t, err)
			require.NotNil(t, uri)

			sipURI, ok := uri.(*SIPURI)
			require.True(t, ok, "Expected SIPURI type")

			assert.Equal(t, tt.expected.Scheme, sipURI.Scheme)
			assert.Equal(t, tt.expected.Host, sipURI.Host)
			assert.Equal(t, tt.expected.User, sipURI.User)
			assert.Equal(t, tt.expected.Password, sipURI.Password)
			assert.Equal(t, tt.expected.Port, sipURI.Port)

			// Test params if they should exist
			if strings.Contains(tt.input, ";") {
				assert.NotNil(t, sipURI.Params)
				if strings.Contains(tt.input, "transport=tcp") {
					transport, exists := sipURI.Params.Get("transport")
					assert.True(t, exists)
					assert.Equal(t, "tcp", transport)
				}
				if strings.Contains(tt.input, "lr") {
					assert.True(t, sipURI.Params.Has("lr"))
				}
			}

			// Test headers if they should exist
			if strings.Contains(tt.input, "?") {
				assert.NotNil(t, sipURI.Headers)
				if strings.Contains(tt.input, "subject=test") {
					subject, exists := sipURI.Headers.Get("subject")
					assert.True(t, exists)
					assert.Equal(t, "test", subject)
				}
				if strings.Contains(tt.input, "priority=urgent") {
					priority, exists := sipURI.Headers.Get("priority")
					assert.True(t, exists)
					assert.Equal(t, "urgent", priority)
				}
			}
		})
	}
}

func TestParseURI_IPv6_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		input       string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "IPv6 address too long",
			input:       "sip:[2001:0db8:85a3:0000:0000:8a2e:0370:7334:extra:too:long]",
			expectError: true,
			errorMsg:    "IPv6 address exceeds maximum length",
		},
		{
			name:        "IPv6 with invalid port after bracket",
			input:       "sip:[2001:db8::1]:abc",
			expectError: true,
			errorMsg:    "invalid port number",
		},
		// {
		// 	name:        "IPv6 nested brackets",
		// 	input:       "sip:[[2001:db8::1]]",
		// 	expectError: true,
		// 	errorMsg:    "nested brackets not allowed",
		// },
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uri, err := ParseURI(tt.input)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, uri)
				// if tt.errorMsg != "" {
				// 	assert.Contains(t, err.Error(), "IPV6")
				// }
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, uri)
			}
		})
	}
}

func TestParseURI_IPv6_RoundTrip(t *testing.T) {
	testCases := []string{
		"sip:[2001:db8::1]",
		"sip:alice@[2001:db8::1]:5060",
		"sip:[::1];transport=tcp",
		"sips:[2001:db8::1]:5061?subject=test",
	}

	for _, input := range testCases {
		t.Run(input, func(t *testing.T) {
			// Parse the URI
			uri, err := ParseURI(input)
			require.NoError(t, err)

			// Convert back to string
			output := uri.String()

			// Parse again
			uri2, err := ParseURI(output)
			require.NoError(t, err)

			// Should be equivalent
			assert.Equal(t, uri.String(), uri2.String())
		})
	}
}

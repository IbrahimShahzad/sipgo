package sip

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type sipUriFSM func(uri *SIPURI, s string) (sipUriFSM, string, error)
type telUriFSM func(uri *TELURI, s string) (telUriFSM, string, error)

func ParseURI(uriStr string) (URI, error) {
	scheme := URISchemeError
	l := min(len(uriStr), 4)
	for i := range l {
		if c := uriStr[i]; c == ':' {
			scheme = detectScheme(uriStr[:i])
		}
	}

	if len(uriStr) < 3 && scheme != URISchemeSIP {
		return nil, fmt.Errorf("not valid uri scheme: %s", uriStr)
	}

	switch scheme {
	case URISchemeSIP, URISchemeSIPS:
		uri := &SIPURI{
			Scheme: scheme,
		}
		if err := ParseSIPURI(uriStr, uri); err != nil {
			return nil, err
		}
		return uri, nil
	case URISchemeTEL:
		uri := &TELURI{
			Scheme: scheme,
		}

		if err := ParseTELURI(uriStr, uri); err != nil {
			return nil, err
		}
		return uri, nil
	default:
		return nil, fmt.Errorf("unsupported uri scheme: %s", uriStr)
	}
}

func ParseTELURI(uriStr string, uri *TELURI) (err error) {
	if len(uriStr) == 0 {
		return errors.New("empty URI")
	}
	state := telStateScheme
	str := uriStr
	for state != nil {
		state, str, err = state(uri, str)
		if err != nil {
			return
		}
	}
	return
}

// ParseSIPURI converts a string representation of a URI into a Uri object.
// Following https://datatracker.ietf.org/doc/html/rfc3261#section-19.1.1
// sip:user:password@host:port;uri-parameters?headers
func ParseSIPURI(uriStr string, uri *SIPURI) (err error) {
	if len(uriStr) == 0 {
		return errors.New("empty URI")
	}
	state := uriStateScheme
	str := uriStr
	for state != nil {
		state, str, err = state(uri, str)
		if err != nil {
			return
		}
	}
	return
}

func uriStateScheme(uri *SIPURI, s string) (sipUriFSM, string, error) {
	// Do fast checks. Minimum uri
	if len(s) < 3 {
		if s == "*" {
			// Normally this goes under url path, but we set on host
			uri.Host = "*"
			uri.Wildcard = true
			return nil, "", nil
		}
		return nil, "", fmt.Errorf("not valid sip uri")
	}

	for i, c := range s {
		if c == ':' {
			scheme := detectScheme(ASCIIToLower(s[:i]))
			if scheme != URISchemeSIP {
				return nil, "", fmt.Errorf("Invalid scheme %d", scheme)
			}
			uri.Scheme = scheme
			return uriStateSlashes, s[i+1:], nil
		}
		// Check is c still ASCII
		if !isASCII(c) {
			return nil, "", fmt.Errorf("invalid uri scheme")
		}
	}

	return nil, "", fmt.Errorf("missing protocol scheme")
}

func uriStateSlashes(uri *SIPURI, s string) (sipUriFSM, string, error) {
	// Check does uri contain slashes
	// They are valid in uri but normally we cut them
	s, uri.HierarhicalSlashes = strings.CutPrefix(s, "//")
	return uriStateUser, s, nil
}

func uriStateUser(uri *SIPURI, s string) (sipUriFSM, string, error) {
	var userend int = 0
	for i, c := range s {
		if c == '[' {
			// IPV6
			return uriStateHost, s[i:], nil
		}

		if c == ':' {
			userend = i
		}

		if c == '@' {
			if userend > 0 {
				uri.User = s[:userend]
				uri.Password = s[userend+1 : i]
			} else {
				uri.User = s[:i]
			}
			return uriStateHost, s[i+1:], nil
		}
	}

	return uriStateHost, s, nil
}

func uriStateHost(uri *SIPURI, s string) (sipUriFSM, string, error) {
	for i, c := range s {
		if c == '[' {
			return uriStateHostIPV6, s[i:], nil
		}

		// TODO this part gets repeated on IPV6
		if c == ':' {
			uri.Host = s[:i]
			return uriStatePort, s[i+1:], nil
		}

		if c == ';' {
			uri.Host = s[:i]
			return uriStateUriParams, s[i+1:], nil
		}

		if c == '?' {
			uri.Host = s[:i]
			return uriStateHeaders, s[i+1:], nil
		}
	}
	// If no special chars found, it means we are at end
	uri.Host = s
	// Check is this wildcard
	uri.Wildcard = s == "*"
	return uriStateUriParams, "", nil
}

func uriStateHostIPV6(uri *SIPURI, s string) (sipUriFSM, string, error) {
	// ffff:ffff:ffff:ffff:ffff:ffff:ffff:ffff max 39 + 2 brackets
	// Do not waste time looking end
	maxs := min(len(s), 42)

	ind := strings.Index(s[:maxs], "]")
	if ind <= 0 {
		return nil, s, fmt.Errorf("IPV6 no closing bracket")
	}
	uri.Host = s[:ind+1]

	if ind+1 == len(s) {
		// finished
		return uriStateUriParams, "", nil
	}

	s = s[ind+1:]

	// Check now termination
	c := s[0]
	if c == ':' {
		return uriStatePort, s[1:], nil
	}

	if c == ';' {
		return uriStateUriParams, s[1:], nil
	}

	if c == '?' {
		return uriStateHeaders, s[1:], nil
	}

	return uriStateUriParams, "", nil
}

func uriStatePort(uri *SIPURI, s string) (sipUriFSM, string, error) {
	var err error
	for i, c := range s {
		if c == ';' {
			uri.Port, err = strconv.Atoi(s[:i])
			return uriStateUriParams, s[i+1:], err
		}

		if c == '?' {
			uri.Port, err = strconv.Atoi(s[:i])
			return uriStateHeaders, s[i+1:], err
		}
	}

	uri.Port, err = strconv.Atoi(s)
	return uriStateUriParams, "", err
}

func uriStateUriParams(uri *SIPURI, s string) (sipUriFSM, string, error) {
	var n int
	var err error
	if len(s) == 0 {
		uri.Params = NewParams()
		uri.Headers = NewParams()
		return nil, s, nil
	}
	uri.Params = NewParams()
	// uri.UriParams, n, err = ParseParams(s, 0, ';', '?', true, true)
	n, err = UnmarshalParams(s, ';', '?', uri.Params)
	if err != nil {
		return nil, s, err
	}

	if n == len(s) {
		n = n - 1
	}

	if s[n] != '?' {
		return nil, s, nil
	}

	return uriStateHeaders, s[n+1:], nil
}

func uriStateHeaders(uri *SIPURI, s string) (sipUriFSM, string, error) {
	var err error
	uri.Headers = NewParams()
	_, err = UnmarshalParams(s, '&', 0, uri.Headers)
	return nil, s, err
}

func telStateScheme(uri *TELURI, s string) (telUriFSM, string, error) {
	// Do fast checks. Minimum uri
	if len(s) < 3 {
		return nil, "", fmt.Errorf("not valid sip uri")
	}

	for i, c := range s {
		if c == ':' {
			uri.Scheme = detectScheme(ASCIIToLower(s[:i]))
			if uri.Scheme != URISchemeTEL {
				return nil, "", fmt.Errorf("not a tel uri sheme")
			}
			return telStateSlashes, s[i+1:], nil
		}
		// Check is c still ASCII
		if !isASCII(c) {
			return nil, "", fmt.Errorf("invalid uri scheme")
		}
	}
	return nil, "", fmt.Errorf("missing protocol scheme")
}

func telStateSlashes(uri *TELURI, s string) (telUriFSM, string, error) {
	// Check does uri contain slashes
	// They are valid in uri but normally we cut them
	s, uri.HierarhicalSlashes = strings.CutPrefix(s, "//")
	return telStateNumber, s, nil
}

func isTelChar(c rune) bool {
	// We allow digits, plus, minus, dot, and some special chars
	return (c >= '0' && c <= '9') || c == '+' || c == '-' || c == '.' || c == '*' || c == '#' || c == ';' || c == '&'
}

func telStateNumber(uri *TELURI, s string) (telUriFSM, string, error) {
	// We expect only digits and some special chars
	for i, c := range s {
		if c == ';' || c == '&' {
			uri.Number = s[:i]
			return telStateParams, s[i+1:], nil
		}
		if !isTelChar(c) {
			return nil, "", fmt.Errorf("invalid tel uri number")
		}
	}
	// If we reach here, we are at the end of the string
	uri.Number = s
	return telStateParams, "", nil
}

func telStateParams(uri *TELURI, s string) (telUriFSM, string, error) {
	var n int
	var err error
	if len(s) == 0 {
		uri.Params = NewParams()
		return nil, s, nil
	}
	uri.Params = NewParams()
	n, err = UnmarshalParams(s, ';', '0', uri.Params)
	if err != nil {
		return nil, s, err
	}

	if n == len(s) {
		n = n - 1
	}

	return nil, s, nil
}

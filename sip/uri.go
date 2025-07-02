package sip

import (
	"io"
	"strconv"
	"strings"
)

type URIScheme int

const (
	URISchemeError URIScheme = iota - 1
	URISchemeSIP
	URISchemeSIPS
	URISchemeTEL
	URISchemeURN
	URISchemeUnknown
)

type URI interface {
	String() string
	GetScheme() string
}

type TELURI struct {
	// Scheme is the scheme of the URI, e.g. "tel"
	Scheme string
	// Wildcard is true if the URI is a wildcard URI, e.g. "tel:*"
	Wildcard bool
	// HierarhicalSlashes is true if the URI has hierarchical slashes, e.g. "tel://"
	HierarhicalSlashes bool
	// User is the user part of the URI, e.g. "1234567890
	Number string
	// Params are the URI parameters, e.g. ";isub=1234;
	Params HeaderParams
	// Headers are the URI headers, e.g. "?header1=value1&header2=value2"
	Headers HeaderParams
}

func (uri *TELURI) String() string {
	var buffer strings.Builder
	uri.StringWrite(&buffer)
	return buffer.String()
}

func (uri *TELURI) GetScheme() string {
	// For backward compatibility. No scheme defaults to tel
	if uri.Scheme == "" {
		return "tel"
	}
	return uri.Scheme
}

// StringWrite writes uri string to buffer
func (uri *TELURI) StringWrite(buffer io.StringWriter) {
	// Normally we expect sip or sips, but it can be tel, urn
	scheme := uri.Scheme
	// For backward compatibility. No scheme defaults to sip
	if uri.Scheme == "" {
		scheme = "tel"
	}

	buffer.WriteString(scheme)
	buffer.WriteString(":")

	if uri.HierarhicalSlashes {
		buffer.WriteString("//")
	}

	if uri.Number != "" {
		buffer.WriteString(uri.Number)
	}

	if (uri.Params != nil) && uri.Params.Length() > 0 {
		buffer.WriteString(";")
		buffer.WriteString(uri.Params.ToString(';'))
	}

}

// SIPURI is parsed form of
// sip:user:password@host:port;uri-parameters?headers
// In case of `sips:“ Encrypted is set to true
type SIPURI struct {
	Scheme string

	// If value is star (*)
	Wildcard bool

	// if // is present
	HierarhicalSlashes bool

	// The user part of the URI: the 'joe' in sip:joe@bloggs.com
	User string

	// The password field of the URI. This is represented in the URI as joe:hunter2@bloggs.com.
	// Note that if a URI has a password field, it *must* have a user field as well.
	// Note that RFC 3261 strongly recommends against the use of password fields in SIP URIs,
	// as they are fundamentally insecure.
	Password string

	// The host part of the URI. This can be a domain, or a string representation of an IP address.
	Host string

	// The port part of the URI. This is optional, and can be empty.
	Port int

	// Any parameters associated with the URI.
	// These are used to provide information about requests that may be constructed from the URI.
	// (For more details, see RFC 3261 section 19.1.1).
	// These appear as a semicolon-separated list of key=value pairs following the host[:port] part.
	Params HeaderParams

	// Any headers to be included on requests constructed from this URI.
	// These appear as a '&'-separated list at the end of the URI, introduced by '?'.
	Headers HeaderParams
}

func (uri *SIPURI) GetScheme() string {
	return uri.Scheme
}

// Generates the string representation of a SipUri struct.
func (uri *SIPURI) String() string {
	var buffer strings.Builder
	uri.StringWrite(&buffer)

	return buffer.String()
}

// StringWrite writes uri string to buffer
func (uri *SIPURI) StringWrite(buffer io.StringWriter) {
	// Normally we expect sip or sips, but it can be tel, urn
	scheme := uri.Scheme
	// For backward compatibility. No scheme defaults to sip
	if uri.Scheme == "" {
		scheme = "sip"
	}

	buffer.WriteString(scheme)
	buffer.WriteString(":")

	if uri.HierarhicalSlashes {
		buffer.WriteString("//")
	}

	// Optional userinfo part.
	if uri.User != "" {
		buffer.WriteString(uri.User)
		if uri.Password != "" {
			buffer.WriteString(":")
			buffer.WriteString(uri.Password)
		}
		buffer.WriteString("@")
	}

	// Compulsory hostname.
	buffer.WriteString(uri.Host)

	// Optional port number.
	if uri.Port > 0 {
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(uri.Port))
	}

	if (uri.Params != nil) && uri.Params.Length() > 0 {
		buffer.WriteString(";")
		buffer.WriteString(uri.Params.ToString(';'))
	}

	if (uri.Headers != nil) && uri.Headers.Length() > 0 {
		buffer.WriteString("?")
		buffer.WriteString(uri.Headers.ToString('&'))
	}
}

// Clone
func (uri *SIPURI) Clone() *SIPURI {
	c := *uri
	if uri.Params != nil {
		c.Params = uri.Params.clone()
	}
	if uri.Headers != nil {
		c.Headers = uri.Headers.clone()
	}
	return &c
}

// IsEncrypted returns true if uri is SIPS uri
func (uri *SIPURI) IsEncrypted() bool {
	return uri.Scheme == "sips"
}

// Endpoint is uri user identifier. user@host[:port]
func (uri *SIPURI) Endpoint() string {
	addr := uri.User + "@" + uri.Host
	if uri.Port > 0 {
		addr += ":" + strconv.Itoa(uri.Port)
	}
	return addr
}

// Addr is uri part without headers and params. sip[s]:user@host[:port]
func (uri *SIPURI) Addr() string {
	scheme := uri.Scheme
	// For backward compatibility. No scheme defaults to sip
	if uri.Scheme == "" {
		scheme = "sip"
	}

	addr := uri.Host
	if uri.User != "" {
		addr = uri.User + "@" + addr
	}
	if uri.Port > 0 {
		addr += ":" + strconv.Itoa(uri.Port)
	}

	if uri.IsEncrypted() {
		return "sips:" + addr
	}
	return scheme + ":" + addr
}

// HostPort represents host:port part
func (uri *SIPURI) HostPort() string {
	p := strconv.Itoa(uri.Port)
	return uri.Host + ":" + p
}

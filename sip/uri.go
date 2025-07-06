package sip

import (
	"errors"
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

var uriSchemes map[URIScheme]string = map[URIScheme]string{
	URISchemeSIP:  "sip",
	URISchemeSIPS: "sips",
	URISchemeTEL:  "tel",
}

func detectScheme(s string) URIScheme {
	switch strings.ToLower(s) {
	case "sip", "*":
		return URISchemeSIP
	case "sips":
		return URISchemeSIPS
	case "tel":
		return URISchemeTEL
	default:
		return URISchemeUnknown
	}
}

type URI interface {
	String() string
	Addr() string
	GetScheme() URIScheme
	GetParams() HeaderParams
	GetUser() string
	GetHost() string
	GetHeaders() HeaderParams
	SetParams(HeaderParams)
	SetHeaders(HeaderParams) error
	SetUser(string)
	SetHost(string) error
	IsWildCard() bool
	IsEncrypted() bool
	Clone() URI
}

type TELURI struct {
	// Scheme is the scheme of the URI, e.g. "tel"
	Scheme URIScheme
	// Wildcard is true if the URI is a wildcard URI, e.g. "tel:*"
	Wildcard bool
	// HierarhicalSlashes is true if the URI has hierarchical slashes, e.g. "tel://"
	HierarhicalSlashes bool
	// User is the user part of the URI, e.g. "1234567890
	Number string
	// Params are the URI parameters, e.g. ";isub=1234;
	Params *HeaderParams
	// Headers are the URI headers, e.g. "?header1=value1&header2=value2"
	Headers *HeaderParams
}

func (u *TELURI) String() string {
	var buffer strings.Builder
	u.StringWrite(&buffer, true)
	return buffer.String()
}

func (u *TELURI) GetScheme() URIScheme {
	return URISchemeTEL
}

func (u *TELURI) GetParams() HeaderParams {
	if u.Params == nil {
		u.Params = NewParams()
	}
	return *u.Params
}

func (u *TELURI) GetHeaders() HeaderParams {
	if u.Headers == nil {
		u.Headers = NewParams()
	}
	return *u.Headers
}

func (u *TELURI) GetUser() string {
	return u.Number
}

func (u *TELURI) GetHost() string {
	return ""
}

func (u *TELURI) Addr() string {
	var buffer strings.Builder
	u.StringWrite(&buffer, false)
	return buffer.String()
}

func (u *TELURI) IsWildCard() bool {
	// no wildcards in tel uri
	return false
}

// StringWrite writes uri string to buffer
func (u *TELURI) StringWrite(buffer io.StringWriter, withParam bool) {
	// Normally we expect sip or sips, but it can be tel, urn

	buffer.WriteString(uriSchemes[u.GetScheme()])
	buffer.WriteString(":")

	if u.HierarhicalSlashes {
		buffer.WriteString("//")
	}

	if u.Number != "" {
		buffer.WriteString(u.Number)
	}

	// in address we do not need to add the params
	if withParam {
		if u.Params.Length() > 0 {
			buffer.WriteString(";")
			buffer.WriteString(u.Params.ToString(';'))
		}
	}
}

// Clone
func (u *TELURI) Clone() URI {
	c := *u
	if u.Params.Length() > 0 {
		c.Params = u.Params.clone()
	}
	return &c
}

func (u *TELURI) IsEncrypted() bool {
	return false
}

func (u *TELURI) TELtoSIP(uri *SIPURI) error {
	domain, ok := u.Params.Get("phone-context")
	if !ok || domain == "" {
		return errors.New("phone-context parameter is required for TEL-URI to SIP conversion")
	}
	u.Params.Remove("phone-context")
	uri.User = removeVisualFromNumber(u.Number)
	// should come from config
	uri.Host = domain
	uri.HierarhicalSlashes = u.HierarhicalSlashes
	uri.Params = u.Params
	uri.Params.Add(Pair{"user-context", "phone"})
	return nil
}

func (u *TELURI) SetParams(p HeaderParams) {
	u.Params = &p
}

func (u *TELURI) SetHeaders(h HeaderParams) error {
	return errors.New("header params not allowed for TEL-URI")
}

func (u *TELURI) SetHost(s string) error {
	return errors.New("host not allowed for TEL-URI")
}

func (u *TELURI) SetUser(s string) {
	u.Number = s
}

// Remove all non-numeric characters from the number
// except for the '+' sign at the beginning
func removeVisualFromNumber(number string) string {
	var sb strings.Builder
	for i, r := range number {
		if i == 0 && r == '+' {
			sb.WriteRune(r)
			continue
		}
		if r >= '0' && r <= '9' {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// SIPURI is parsed form of
// sip:user:password@host:port;uri-parameters?headers
// In case of `sips:“ Encrypted is set to true
type SIPURI struct {
	Scheme URIScheme

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
	Params *HeaderParams

	// Any headers to be included on requests constructed from this URI.
	// These appear as a '&'-separated list at the end of the URI, introduced by '?'.
	Headers *HeaderParams
}

func (u *SIPURI) GetScheme() URIScheme {
	return u.Scheme
}

// Generates the string representation of a SipUri struct.
func (u *SIPURI) String() string {
	var buffer strings.Builder
	u.StringWrite(&buffer)

	return buffer.String()
}

// StringWrite writes uri string to buffer
func (u *SIPURI) StringWrite(buffer io.StringWriter) {

	buffer.WriteString(uriSchemes[u.GetScheme()])
	buffer.WriteString(":")

	if u.HierarhicalSlashes {
		buffer.WriteString("//")
	}

	// Optional userinfo part.
	if u.User != "" {
		buffer.WriteString(u.User)
		if u.Password != "" {
			buffer.WriteString(":")
			buffer.WriteString(u.Password)
		}
		buffer.WriteString("@")
	}

	// Compulsory hostname.
	buffer.WriteString(u.Host)

	// Optional port number.
	if u.Port > 0 {
		buffer.WriteString(":")
		buffer.WriteString(strconv.Itoa(u.Port))
	}

	if u.Params != nil && u.Params.Length() > 0 {
		buffer.WriteString(";")
		buffer.WriteString(u.Params.ToString(';'))
	}

	if u.Headers != nil && u.Headers.Length() > 0 {
		buffer.WriteString("?")
		buffer.WriteString(u.Headers.ToString('&'))
	}
}

// Clone
func (u *SIPURI) Clone() URI {
	c := *u
	if u.Params.Length() > 0 {
		c.Params = u.Params.clone()
	}
	if u.Headers.Length() > 0 {
		c.Headers = u.Headers.clone()
	}
	return &c
}

// IsEncrypted returns true if uri is SIPS uri
func (u *SIPURI) IsEncrypted() bool {
	return u.Scheme == URISchemeSIPS
}

// Endpoint is uri user identifier. user@host[:port]
func (u *SIPURI) Endpoint() string {
	addr := u.User + "@" + u.Host
	if u.Port > 0 {
		addr += ":" + strconv.Itoa(u.Port)
	}
	return addr
}

// Addr is uri part without headers and params. sip[s]:user@host[:port]
func (u *SIPURI) Addr() string {
	scheme := uriSchemes[u.Scheme]
	// For backward compatibility. No scheme defaults to sip

	addr := u.Host
	if u.User != "" {
		addr = u.User + "@" + addr
	}
	if u.Port > 0 {
		addr += ":" + strconv.Itoa(u.Port)
	}

	if u.IsEncrypted() {
		return "sips:" + addr
	}
	return scheme + ":" + addr
}

// HostPort represents host:port part
func (u *SIPURI) HostPort() string {
	p := strconv.Itoa(u.Port)
	return u.Host + ":" + p
}

func (u *SIPURI) GetParams() HeaderParams {
	if u.Params == nil {
		u.Params = NewParams()
	}
	return *u.Params
}

func (u *SIPURI) GetHeaders() HeaderParams {
	if u.Headers == nil {
		u.Headers = NewParams()
	}
	return *u.Headers
}

func (u *SIPURI) GetHost() string {
	return u.Host
}

func (u *SIPURI) GetUser() string {
	return u.User
}

func (u *SIPURI) IsWildCard() bool {
	return u.Wildcard
}

func (u *SIPURI) SetParams(p HeaderParams) {
	u.Params = &p
}

func (u *SIPURI) SetHeaders(h HeaderParams) error {
	u.Headers = &h
	return nil
}

func (u *SIPURI) SetHost(s string) error {
	if s == "*" {
		u.Wildcard = true
	}
	u.Host = s
	return nil
}

func (u *SIPURI) SetUser(s string) {
	u.User = s
}

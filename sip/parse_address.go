package sip

import (
	"errors"
	"fmt"
	"strings"
)

var (
	errEmptyAddress = errors.New("empty Address")
	errNoURIPresent = errors.New("no URI present")
	errInvalidUri   = errors.New("invalid uri, missing end bracket")
)

type NameAddress struct {
	DisplayName string
	URI         URI
	Params      *HeaderParams
}

type addressFSM func(dispName *NameAddress, s string) (addressFSM, string, error)

// ParseAddressValue parses an address - such as from a From, To, or
// Contact header. It returns:
// See RFC 3261 section 20.10 for details on parsing an address.
func ParseAddressValue(addressText string) (address *NameAddress, err error) {
	if len(addressText) == 0 {
		return nil, errEmptyAddress
	}

	// adds alloc but easier to maintain
	a := NameAddress{
		Params: NewParams(),
	}

	err = parseNameAddress(addressText, &a)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

// parseNameAddress
// name-addr      =  [ display-name ] LAQUOT addr-spec RAQUOT
// addr-spec      =  SIP-URI / SIPS-URI / absoluteURI
// TODO Consider exporting this
func parseNameAddress(addressText string, a *NameAddress) (err error) {
	state := addressStateDisplayName
	str := addressText
	for state != nil {
		state, str, err = state(a, str)
		if err != nil {
			return
		}
	}
	return nil
}

func addressStateDisplayName(a *NameAddress, s string) (addressFSM, string, error) {
	for i, c := range s {
		if c == '"' {
			return addressStateDisplayNameQuoted, s[i+1:], nil
		}

		// https://datatracker.ietf.org/doc/html/rfc3261#section-20.10
		// When the header field value contains a display name, the URI
		// including all URI parameters is enclosed in "<" and ">".  If no "<"
		// and ">" are present, all parameters after the URI are header
		// parameters, not URI parameters.
		if c == '<' {
			a.DisplayName = strings.TrimSpace(s[:i])
			return addressStateUriBracket, s[i+1:], nil
		}

		if c == ';' {
			// detect early
			// uri can be without <> in that case there all after ; are header params
			return addressStateUri, s, nil
		}
	}

	// No DisplayName found
	return addressStateUri, s, nil
}

func addressStateDisplayNameQuoted(a *NameAddress, s string) (addressFSM, string, error) {
	var escaped bool
	for i, c := range s {
		if c == '\\' {
			// https://datatracker.ietf.org/doc/html/rfc3261#section-25.1
			// The backslash character ("\") MAY be used as a single-character
			// quoting mechanism only within quoted-string and comment constructs.
			escaped = !escaped
			continue
		}

		if escaped {
			if c == 0xA || c == 0x0D {
				// quoted-pair  =  "\" (%x00-09 / %x0B-0C / %x0E-7F)
				return nil, s, fmt.Errorf("invalid display name, not allowed to escape '0x%02X' in '%s'", c, s)
			}
			escaped = false
			continue
		}

		if c == '"' {
			a.DisplayName = s[:i]
			s = s[i+1:]
			for i, c := range s {
				if c == '<' {
					return addressStateUriBracket, s[i+1:], nil
				}

				if c == ';' {
					return addressStateUri, s[i+1:], nil
				}
			}
			return nil, s, fmt.Errorf("no uri after display name")
		}
	}

	return nil, s, fmt.Errorf("invalid uri display name inside quotes")
}

func addressStateUriBracket(a *NameAddress, s string) (addressFSM, string, error) {
	if len(s) == 0 {
		return nil, s, errNoURIPresent
	}

	for i, c := range s {
		if c == '>' {
			var err error
			a.URI, err = ParseURI(s[:i])
			return addressStateHeaderParams, s[i+1:], err
		}
	}
	return nil, s, errInvalidUri
}

func addressStateUri(a *NameAddress, s string) (addressFSM, string, error) {
	if len(s) == 0 {
		return nil, s, errors.New("no URI present")
	}

	for i, c := range s {
		if c == '*' {
			a.URI = &SIPURI{
				Scheme:   URISchemeSIP,
				Host:     "*",
				Wildcard: true,
			}
			return nil, "", nil
		}

		if c == ';' {
			var err error
			a.URI, err = ParseURI(s[:i])
			return addressStateHeaderParams, s[i+1:], err
		}

	}

	// No header params detected
	var err error
	a.URI, err = ParseURI(s)
	return nil, s, err
}

func addressStateHeaderParams(a *NameAddress, s string) (addressFSM, string, error) {

	addParam := func(equal int, s string) {

		if equal > 0 {
			name := s[:equal]
			val := s[equal+1:]
			a.Params.Add(Pair{name, val})
			return
		}

		if len(s) == 0 {
			// could be just ;
			return
		}

		// Case when we have key name but not value. ex ;+siptag;
		name := s[:]
		a.Params.Add(Pair{name, ""})
	}

	equal := -1
	for i, c := range s {
		if c == '=' {
			equal = i
			continue
		}

		if c == ';' {
			addParam(equal, s[:i])
			return addressStateHeaderParams, s[i+1:], nil
		}
	}

	addParam(equal, s)
	return nil, s, nil
}

// headerParserTo generates ToHeader
func headerParserTo(headerName string, headerText string) (header Header, err error) {
	h := &ToHeader{}
	return h, parseToHeader(headerText, h)
}

func parseToHeader(headerText string, h *ToHeader) error {
	var err error
	// params := NewParams()
	h.NameAddress, err = ParseAddressValue(headerText)
	if err != nil {
		return err
	}

	if h.URI == nil {
		return errors.New("the parsed address is nil")
	}

	if h.URI.IsWildCard() {
		// The Wildcard '*' URI is only permitted in Contact headers.
		err = fmt.Errorf(
			"wildcard uri not permitted in to: header: %s",
			headerText,
		)
		return err
	}
	return nil
}

// headerParserFrom generates FromHeader
func headerParserFrom(headerName string, headerText string) (header Header, err error) {
	h := &FromHeader{}
	return h, parseFromHeader(headerText, h)
}

func parseFromHeader(headerText string, h *FromHeader) error {
	var err error

	// h.DisplayName, err = ParseAddressValue(headerText, h.Address, h.Params)
	// params := NewParams()
	h.NameAddress, err = ParseAddressValue(headerText)
	// h.DisplayName, h.Address, h.Params, err = ParseAddressValue(headerText)
	if err != nil {
		return err
	}

	if h.URI == nil {
		return errors.New("could not parse address")
	}

	if h.URI.IsWildCard() {
		// The Wildcard '*' URI is only permitted in Contact headers.
		err = fmt.Errorf(
			"wildcard uri not permitted in to: header: %s",
			headerText,
		)
		return err
	}
	return nil
}

func headerParserContact(headerName string, headerText string) (header Header, err error) {
	h := ContactHeader{
		NameAddress: &NameAddress{
			Params: NewParams(),
		},
	}
	return &h, parseContactHeader(headerText, &h)
}

// parseContactHeader generates ContactHeader
func parseContactHeader(headerText string, h *ContactHeader) error {
	inBrackets := false
	inQuotes := false

	endInd := len(headerText)
	end := endInd - 1

	var err error
	for idx, char := range headerText {
		if char == '<' && !inQuotes {
			inBrackets = true
		} else if char == '>' && !inQuotes {
			inBrackets = false
		} else if char == '"' {
			inQuotes = !inQuotes
		} else if !inQuotes && !inBrackets {
			switch {
			case char == ',':
				err = errComaDetected(idx)
			case idx == end:
				endInd = idx + 1
			default:
				continue
			}

			break
		}
	}

	if h == nil {
		panic("ContactHeader is nil")
	}

	var e error
	// params := NewParams()
	h.NameAddress, e = ParseAddressValue(headerText[:endInd])
	if e != nil {
		return e
	}

	return err
}

func headerParserRoute(headerName string, headerText string) (header Header, err error) {
	// Append a comma to simplify the parsing code; we split address sections
	// on commas, so use a comma to signify the end of the final address section.
	h := RouteHeader{}
	return &h, parseRouteHeader(headerText, &h)
}

// parseRouteHeader parser RouteHeader
func parseRouteHeader(headerText string, h *RouteHeader) error {
	return parseRouteAddress(headerText, &h.Address)
}

// parseRouteHeader generates RecordRouteHeader
func headerParserRecordRoute(headerName string, headerText string) (header Header, err error) {
	// Append a comma to simplify the parsing code; we split address sections
	// on commas, so use a comma to signify the end of the final address section.
	h := RecordRouteHeader{}
	return &h, parseRecordRouteHeader(headerText, &h)
}

func parseRecordRouteHeader(headerText string, h *RecordRouteHeader) error {
	return parseRouteAddress(headerText, &h.Address)
}

func headerParserReferTo(headerName string, headerText string) (header Header, err error) {
	h := ReferToHeader{}
	return &h, parseReferToHeader(headerText, &h)
}

func parseReferToHeader(headerText string, h *ReferToHeader) error {
	return parseRouteAddress(headerText, &h.Address) // calling parseRouteAddress because the structure is same
}

func headerParserReferredBy(headerName string, headerText string) (header Header, err error) {
	h := &ReferredByHeader{}
	return h, parseReferredByHeader(headerText, h)
}

func parseReferredByHeader(headerText string, h *ReferredByHeader) error {
	var err error

	// h.Params = NewParams()
	h.NameAddress, err = ParseAddressValue(headerText)
	if err != nil {
		return err
	}

	if h.URI.IsWildCard() {
		// The Wildcard '*' URI is only permitted in Contact headers.
		err = fmt.Errorf(
			"wildcard uri not permitted in to: header: %s",
			headerText,
		)
		return err
	}
	return nil
}

func parseRouteAddress(headerText string, address *SIPURI) (err error) {
	inBrackets := false
	inQuotes := false
	end := len(headerText) - 1
	for idx, char := range headerText {
		if char == '<' && !inQuotes {
			inBrackets = true
			continue
		}
		if char == '>' && !inQuotes {
			inBrackets = false
		} else if char == '"' {
			inQuotes = !inQuotes
		}

		if !inQuotes && !inBrackets {
			switch {
			case char == ',':
				err = errComaDetected(idx)
			case idx == end:
				idx = idx + 1
			default:
				continue
			}

			// params := NewParams()
			nameAddress, e := ParseAddressValue(headerText[:idx])
			if e != nil {
				return e
			}
			// should be a SIP
			addr, ok := nameAddress.URI.(*SIPURI)
			if !ok {
				return fmt.Errorf("expected SIPURI, got %T", nameAddress.URI)
			}
			*address = *addr
			break
		}
	}
	return
}

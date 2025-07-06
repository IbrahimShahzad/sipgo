package sip

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseAddressValue(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		address := "\"Bob\" <sips:bob:password@127.0.0.1:5060;user=phone>;tag=1234"

		uri := SIPURI{}

		addr, err := ParseAddressValue(address)
		suri, ok := addr.URI.(*SIPURI)
		if !ok {
			t.Errorf("could not convert to sip %+v", addr)
			t.FailNow()
		}
		uri = *suri

		assert.Nil(t, err)
		assert.Equal(t, "sips:bob:password@127.0.0.1:5060;user=phone", uri.String())
		assert.Equal(t, "tag=1234", addr.Params.String())

		assert.Equal(t, "Bob", addr.DisplayName)
		assert.Equal(t, "bob", uri.User)
		assert.Equal(t, "password", uri.Password)
		assert.Equal(t, "127.0.0.1", uri.Host)
		assert.Equal(t, 5060, uri.Port)
		assert.Equal(t, true, uri.IsEncrypted())
		assert.Equal(t, false, uri.Wildcard)

		user, ok := uri.Params.Get("user")
		assert.True(t, ok)
		assert.Equal(t, 1, uri.Params.Length())
		assert.Equal(t, "phone", user)

	})

	t.Run("no display name", func(t *testing.T) {
		address := "sip:1215174826@222.222.222.222;tag=9300025590389559597"
		uri := SIPURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		require.NoError(t, err)
		suri, ok := addr.URI.(*SIPURI)
		if !ok {
			t.Errorf("could not convert to SIPURI")
			t.FailNow()
		}
		uri = *suri

		assert.Equal(t, "", addr.DisplayName)
		assert.Equal(t, "1215174826", uri.User)
		assert.Equal(t, "222.222.222.222", uri.Host)
		assert.Equal(t, false, uri.IsEncrypted())
	})

	t.Run("nil uri params", func(t *testing.T) {
		address := "sip:1215174826@222.222.222.222:5066"
		uri := SIPURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		require.NoError(t, err)
		suri, ok := addr.URI.(*SIPURI)
		if !ok {
			t.Errorf("could not convert to SIPURI")
			t.FailNow()
		}
		uri = *suri

		assert.Equal(t, "", addr.DisplayName)
		assert.Equal(t, "1215174826", uri.User)
		assert.Equal(t, "222.222.222.222", uri.Host)
		assert.Equal(t, &HeaderParams{
			keys: make(map[string]int),
			data: []Pair{},
		}, uri.Params)
		assert.Equal(t, false, uri.IsEncrypted())
	})

	t.Run("wildcard", func(t *testing.T) {
		address := "*"
		uri := SIPURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		require.NoError(t, err)
		suri, ok := addr.URI.(*SIPURI)
		if !ok {
			t.Errorf("could not convert to SIPURI")
			t.FailNow()
		}
		uri = *suri
		assert.Equal(t, "", addr.DisplayName)
		assert.Equal(t, "*", uri.Host)
		assert.Equal(t, true, uri.Wildcard)
	})

	t.Run("quoted-pairs", func(t *testing.T) {
		address := "\"!\\\"#$%&/'()*+-.,0123456789:;<=>? @ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\\\]^_'abcdefghijklmnopqrstuvwxyz{|}\" <sip:bob@127.0.0.1:5060;user=phone>;tag=1234"
		uri := SIPURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		sUri, ok := addr.URI.(*SIPURI)
		if !ok {
			t.Errorf("could not convert to SIP URI")
		}
		uri = *sUri
		require.NoError(t, err)

		assert.Equal(t, "sip:bob@127.0.0.1:5060;user=phone", uri.String())
		assert.Equal(t, "tag=1234", addr.Params.String())

		assert.Equal(t, "!\\\"#$%&/'()*+-.,0123456789:;<=>? @ABCDEFGHIJKLMNOPQRSTUVWXYZ[\\\\]^_'abcdefghijklmnopqrstuvwxyz{|}", addr.DisplayName)
		assert.Equal(t, "bob", uri.User)
		assert.Equal(t, "", uri.Password)
		assert.Equal(t, "127.0.0.1", uri.Host)
		assert.Equal(t, 5060, uri.Port)
		assert.Equal(t, false, uri.IsEncrypted())
		assert.Equal(t, false, uri.Wildcard)

		user, ok := uri.Params.Get("user")
		assert.True(t, ok)
		assert.Equal(t, 1, uri.Params.Length())
		assert.Equal(t, "phone", user)

	})

	t.Run("tel uri", func(t *testing.T) {
		address := "tel:1215174826"
		uri := TELURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		require.NoError(t, err)
		suri, ok := addr.URI.(*TELURI)
		if !ok {
			t.Errorf("could not convert to TELURI")
			t.FailNow()
		}
		uri = *suri

		assert.Equal(t, "", addr.DisplayName)
		assert.Equal(t, "1215174826", uri.Number)
		assert.Equal(t, URISchemeTEL, uri.Scheme)
		assert.Equal(t, NewParams(), uri.Params)
		assert.Equal(t, false, uri.IsEncrypted())
	})

	t.Run("tel uri with global", func(t *testing.T) {
		address := "tel:+1215174826"
		uri := TELURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		require.NoError(t, err)
		suri, ok := addr.URI.(*TELURI)
		if !ok {
			t.Errorf("could not convert to TELURI")
			t.FailNow()
		}
		uri = *suri

		assert.Equal(t, "", addr.DisplayName)
		assert.Equal(t, "+1215174826", uri.Number)
		assert.Equal(t, URISchemeTEL, uri.Scheme)
		assert.Equal(t, NewParams(), uri.Params)
		assert.Equal(t, false, uri.IsEncrypted())
	})

	t.Run("tel uri with visual identifiers", func(t *testing.T) {
		address := "tel:+121-517-4826"
		uri := TELURI{}
		// params := NewParams()
		addr, err := ParseAddressValue(address)
		require.NoError(t, err)
		suri, ok := addr.URI.(*TELURI)
		if !ok {
			t.Errorf("could not convert to TELURI")
			t.FailNow()
		}
		uri = *suri

		assert.Equal(t, "", addr.DisplayName)
		assert.Equal(t, "+121-517-4826", uri.Number)
		assert.Equal(t, URISchemeTEL, uri.Scheme)
		assert.Equal(t, NewParams(), uri.Params)
		assert.Equal(t, false, uri.IsEncrypted())
	})

}

func TestParseAddressBad(t *testing.T) {

	t.Run("double ports in uri", func(t *testing.T) {
		address := "<sip:127.0.0.1:5060:5060;lr;transport=udp>"
		_, err := ParseAddressValue(address)
		require.Error(t, err)
	})
}

// TODO
// func TestParseAddressMultiline(t *testing.T) {
// contact:
// 	+`Contact: "Mr. Watson" <sip:watson@worcester.bell-telephone.com>
// 	;q=0.7; expires=3600,
// 	"Mr. Watson" <mailto:watson@bell-telephone.com> ;q=0.1`
// }

func BenchmarkParseAddress(b *testing.B) {
	address := "\"Bob\" <sips:bob:password@127.0.0.1:5060;user=phone>;tag=1234"

	for b.Loop() {
		nameAddress, err := ParseAddressValue(address)
		_, ok := nameAddress.URI.(*SIPURI)
		if !ok {
			b.Errorf("could not convert to SIPURI")
			b.FailNow()
		}
		assert.Nil(b, err)
		assert.Equal(b, "Bob", nameAddress.DisplayName)
	}
}

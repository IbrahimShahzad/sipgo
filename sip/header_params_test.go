package sip

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSepToString(t *testing.T) {
	hp := NewParams()
	hp.Add(
		Pair{"tag", "aaa"},
		Pair{"branch", "bbb"},
	)

	for _, sep := range []uint8{';', '&', '?'} {
		str := hp.ToString(sep)
		arr := strings.Split(str, string(sep))
		assert.Equal(t, strings.Join(arr, string(sep)), str)
	}
}

func TestAddRemoveParams(t *testing.T) {
	hp := NewParams()
	hp.Add(
		Pair{"tag", "aaa"},
		Pair{"branch", "bbb"},
	)

	assert.Equal(t, 2, hp.Length())
	tag, exists := hp.Get("tag")
	assert.True(t, exists)
	assert.Equal(t, "aaa", tag)
	branch, exists := hp.Get("branch")
	assert.True(t, exists)
	assert.Equal(t, "bbb", branch)

	hp.Add(Pair{"tag", "ccc"})
	assert.Equal(t, 2, hp.Length())
	// Check that the value of tag was updated
	tag, exists = hp.Get("tag")
	assert.True(t, exists)
	assert.Equal(t, "ccc", tag)

	hp.Remove("branch")
	branch, exists = hp.Get("branch")
	assert.False(t, exists)
	assert.Equal(t, "", branch)
	assert.Equal(t, 1, hp.Length())
}

func TestHeaderParams(t *testing.T) {
	hp := NewParams()

	t.Run("Keys", func(t *testing.T) {
		hp.Add(
			Pair{"tag", "aaa"},
			Pair{"branch", "bbb"},
		)
		assert.Equal(t, 2, hp.Length())
		tag, exists := hp.Get("tag")
		assert.True(t, exists)
		assert.Equal(t, "aaa", tag)
		branch, exists := hp.Get("branch")
		assert.True(t, exists)
		assert.Equal(t, "bbb", branch)
		keys := hp.Keys()
		// The order of keys should be preserved
		assert.Equal(t, 2, len(keys))
		assert.Equal(t, "tag", keys[0])
		assert.Equal(t, "branch", keys[1])
	})
}
func TestHeaderParamsAdd(t *testing.T) {
	t.Run("Add single pair to empty params", func(t *testing.T) {
		hp := &HeaderParams{}
		hp.Add(Pair{"foo", "bar"})
		val, ok := hp.Get("foo")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)
		assert.Equal(t, 1, hp.Length())
		assert.Equal(t, []string{"foo"}, hp.Keys())
	})

	t.Run("Add multiple pairs at once", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"a", "1"}, Pair{"b", "2"}, Pair{"c", "3"})
		assert.Equal(t, 3, hp.Length())
		assert.Equal(t, "1", mustGet(t, hp, "a"))
		assert.Equal(t, "2", mustGet(t, hp, "b"))
		assert.Equal(t, "3", mustGet(t, hp, "c"))
		assert.Equal(t, []string{"a", "b", "c"}, hp.Keys())
	})

	t.Run("Overwrite existing key", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"x", "y"})
		hp.Add(Pair{"x", "z"})
		val, ok := hp.Get("x")
		assert.True(t, ok)
		assert.Equal(t, "z", val)
		assert.Equal(t, 1, hp.Length())
		assert.Equal(t, []string{"x"}, hp.Keys())
	})

	t.Run("Add after overwrite preserves order", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"a", "1"}, Pair{"b", "2"})
		hp.Add(Pair{"a", "3"})
		assert.Equal(t, []string{"a", "b"}, hp.Keys())
		assert.Equal(t, "3", mustGet(t, hp, "a"))
	})

	t.Run("Add to nil HeaderParams", func(t *testing.T) {
		var hp *HeaderParams
		hp = hp.Add(Pair{"foo", "bar"})
		assert.NotNil(t, hp)
		val, ok := hp.Get("foo")
		assert.True(t, ok)
		assert.Equal(t, "bar", val)
	})

	t.Run("Add to HeaderParams with nil maps", func(t *testing.T) {
		hp := &HeaderParams{}
		hp.Add(Pair{"x", "y"})
		val, ok := hp.Get("x")
		assert.True(t, ok)
		assert.Equal(t, "y", val)
	})

	t.Run("Add with empty value", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"empty", ""})
		val, ok := hp.Get("empty")
		assert.True(t, ok)
		assert.Equal(t, "", val)
		assert.Equal(t, 1, hp.Length())
	})

	t.Run("Add duplicate keys in one call", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"dup", "1"}, Pair{"dup", "2"})
		val, ok := hp.Get("dup")
		assert.True(t, ok)
		assert.Equal(t, "2", val)
		assert.Equal(t, 1, hp.Length())
	})
}

func TestHeaderParamsHas(t *testing.T) {
	t.Run("Has returns true for existing key", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"foo", "bar"})
		assert.True(t, hp.Has("foo"))
	})

	t.Run("Has returns false for non-existing key", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"foo", "bar"})
		assert.False(t, hp.Has("baz"))
	})

	t.Run("Has returns false for empty HeaderParams", func(t *testing.T) {
		hp := NewParams()
		assert.False(t, hp.Has("foo"))
	})

	t.Run("Has returns false after Remove", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"foo", "bar"})
		hp.Remove("foo")
		assert.False(t, hp.Has("foo"))
	})

	t.Run("Has works with multiple keys", func(t *testing.T) {
		hp := NewParams()
		hp.Add(Pair{"a", "1"}, Pair{"b", "2"}, Pair{"c", "3"})
		assert.True(t, hp.Has("a"))
		assert.True(t, hp.Has("b"))
		assert.True(t, hp.Has("c"))
		assert.False(t, hp.Has("d"))
	})
}

func TestHeaderParamsEquals(t *testing.T) {
	t.Run("Equal empty params", func(t *testing.T) {
		hp1 := NewParams()
		hp2 := NewParams()
		assert.True(t, hp1.Equals(hp2))
	})

	t.Run("Equal nil params", func(t *testing.T) {
		hp1 := NewParams()
		assert.False(t, hp1.Equals(nil))
	})

	t.Run("Equal single param", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"foo", "bar"})
		hp2 := NewParams().Add(Pair{"foo", "bar"})
		assert.True(t, hp1.Equals(hp2))
	})

	t.Run("Equal multiple params, same order", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"a", "1"}, Pair{"b", "2"})
		hp2 := NewParams().Add(Pair{"a", "1"}, Pair{"b", "2"})
		assert.True(t, hp1.Equals(hp2))
	})

	t.Run("Equal multiple params, different order", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"a", "1"}, Pair{"b", "2"})
		hp2 := NewParams().Add(Pair{"b", "2"}, Pair{"a", "1"})
		// Order matters for Equals, so this should be false
		assert.False(t, hp1.Equals(hp2))
	})

	t.Run("Different values for same key", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"foo", "bar"})
		hp2 := NewParams().Add(Pair{"foo", "baz"})
		assert.False(t, hp1.Equals(hp2))
	})

	t.Run("Different keys", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"foo", "bar"})
		hp2 := NewParams().Add(Pair{"baz", "bar"})
		assert.False(t, hp1.Equals(hp2))
	})

	t.Run("Different number of params", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"foo", "bar"})
		hp2 := NewParams().Add(Pair{"foo", "bar"}, Pair{"baz", "qux"})
		assert.False(t, hp1.Equals(hp2))
	})

	t.Run("Equal after Remove", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"foo", "bar"}, Pair{"baz", "qux"})
		hp2 := NewParams().Add(Pair{"foo", "bar"}, Pair{"baz", "qux"})
		hp1.Remove("baz")
		hp2.Remove("baz")
		assert.True(t, hp1.Equals(hp2))
	})

	t.Run("Equal with empty values", func(t *testing.T) {
		hp1 := NewParams().Add(Pair{"foo", ""})
		hp2 := NewParams().Add(Pair{"foo", ""})
		assert.True(t, hp1.Equals(hp2))
	})
}

func mustGet(t *testing.T, hp *HeaderParams, key string) string {
	val, ok := hp.Get(key)
	if !ok {
		t.Fatalf("key %q not found", key)
	}
	return val
}
func BenchmarkHeaderParams(b *testing.B) {

	testParams := func(b *testing.B, hp *HeaderParams) {
		hp = hp.Add(
			Pair{"branch", "assadkjkgeijdas"},
			Pair{"received", "127.0.0.1"},
			Pair{"toremove", "removeme"})
		hp = hp.Remove("toremove")

		if _, exists := hp.Get("received"); !exists {
			b.Fatal("received does not exists")
		}

		s := hp.ToString(';')
		if len(s) == 0 {
			b.Fatal("Params empty")
		}

		if s != "branch=assadkjkgeijdas;received=127.0.0.1" && s != "received=127.0.0.1;branch=assadkjkgeijdas" {
			b.Fatal("Bad parsing")
		}
	}

	// Lot of allocations makes slower parsing
	// b.Run("GOSIP", func(b *testing.B) {
	// 	for i := 0; i < b.N; i++ {
	// 		hp := NewParams()
	// 		testParams(b, hp.(Params))
	// 	}
	// })

	// Our version must be faster than GOSIP
	b.Run("MAP", func(b *testing.B) {
		for b.Loop() {
			hp := NewParams()
			testParams(b, hp)
		}
	})

}

func BenchmarkStringConcetationVsBuffer(b *testing.B) {
	name := "Callid"
	value := "abcdefge1234566"
	// expected := name + ":" + value
	b.ResetTimer()

	b.Run("Concat", func(b *testing.B) {
		var buf strings.Builder
		for i := 0; i < b.N; i++ {
			buf.WriteString(name + ":" + value)
		}
		if buf.Len() == 0 {
			b.FailNow()
		}
	})

	// Our version must be faster than GOSIP
	b.Run("Buffer", func(b *testing.B) {
		var buf strings.Builder
		for i := 0; i < b.N; i++ {
			buf.WriteString(name)
			buf.WriteString(":")
			buf.WriteString(value)
			// if buf.String() != expected {
			// 	b.FailNow()
			// }
		}
		if buf.Len() == 0 {
			b.FailNow()
		}
	})
}

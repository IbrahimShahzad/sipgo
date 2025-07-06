package sip

import (
	"io"
	"strings"
)

// key-value pair
type Pair struct {
	Key string
	Val string
}

type HeaderParams struct {
	keys map[string]int // to store keys and their order
	data []Pair         // to store key-value pairs
}

// Create an empty set of parameters.
func NewParams() *HeaderParams {
	return &HeaderParams{
		keys: make(map[string]int),
		data: make([]Pair, 0),
	}
}

// Keys return a slice of keys, in order.
func (hp HeaderParams) Keys() []string {
	k := make([]string, len(hp.keys))
	// order by index
	for key, i := range hp.keys {
		k[i] = key
	}
	return k
}

// Get returns existing key
func (hp HeaderParams) Get(key string) (string, bool) {
	v, ok := hp.keys[key]
	if !ok {
		return "", false
	}
	val := hp.data[v].Val
	return val, ok
}

// Add adds one or more key-value pairs to params.
// If a key already exists, its value will be overwritten.
func (hp *HeaderParams) Add(pair ...Pair) *HeaderParams {
	if hp == nil {
		newHp := NewParams()
		return newHp.Add(pair...)
	}

	if hp.keys == nil || hp.data == nil {
		const defaultCapacity = 4
		capacity := max(len(pair), defaultCapacity)
		hp.keys = make(map[string]int, capacity)
		hp.data = make([]Pair, 0, capacity)
	}

	for _, p := range pair {
		if i, exists := hp.keys[p.Key]; exists {
			hp.data[i].Val = p.Val
			continue
		}
		hp.keys[p.Key] = len(hp.data)
		hp.data = append(hp.data, Pair{Key: p.Key, Val: p.Val})
	}
	return hp
}

// Remove removes param with exact key
func (hp *HeaderParams) Remove(key string) *HeaderParams {
	// remove key from keys map
	if i, exists := hp.keys[key]; exists {
		hp.data = append(hp.data[:i], hp.data[i+1:]...)
		delete(hp.keys, key)
		for k, v := range hp.keys {
			if v > i {
				hp.keys[k] = v - 1
			}
		}
	}
	return hp
}

// Has checks does key exists
func (hp HeaderParams) Has(key string) bool {
	_, exists := hp.keys[key]
	return exists
}

// Clone returns underneath map copied
func (hp HeaderParams) Clone() *HeaderParams {
	return hp.clone()
}

func (hp HeaderParams) clone() *HeaderParams {
	dup := NewParams()
	dup.keys = make(map[string]int, len(hp.keys))
	dup.data = make([]Pair, 0)
	// maintain order of keys
	for i, v := range hp.data {
		dup.data = append(dup.data, Pair{Key: v.Key, Val: v.Val})
		dup.keys[v.Key] = i
	}
	return dup
}

// ToString renders params to a string.
// Note that this does not escape special characters, this should already have been done before calling this method.
func (hp HeaderParams) ToString(sep uint8) string {
	if len(hp.data) == 0 {
		return ""
	}

	sepstr := string(sep)
	var buffer strings.Builder

	for _, p := range hp.data {
		buffer.WriteString(sepstr)
		buffer.WriteString(p.Key)

		if p.Val != "" {
			// Params can be without value like ;lr;
			buffer.WriteString("=")
			buffer.WriteString(p.Val)
		}
	}

	return buffer.String()[1:]
}

// ToStringWrite is same as ToString but it stores to defined buffer instead returning string
func (hp HeaderParams) ToStringWrite(sep uint8, buffer io.StringWriter) {
	if len(hp.data) == 0 {
		return
	}

	// sepstr := fmt.Sprintf("%c", sep)
	sepstr := string(sep)
	i := 0
	for _, p := range hp.data {
		if i > 0 {
			buffer.WriteString(sepstr)
		}
		i++

		buffer.WriteString(p.Key)
		if p.Val == "" {
			continue
		}

		if p.Val != "" {
			buffer.WriteString("=")
			buffer.WriteString(p.Val)
		}
	}
}

// String returns params joined with '&' char.
func (hp HeaderParams) String() string {
	return hp.ToString('&')
}

// Length returns number of params.
func (hp HeaderParams) Length() int {
	return len(hp.data)
}

// Equals checks if two HeaderParams are equal.
// Two HeaderParams are equal if they have the same keys and values, in the same order.
func (hp HeaderParams) Equals(o *HeaderParams) bool {
	if o == nil {
		return false
	}

	hplen := hp.Length()
	qlen := o.Length()
	if hplen != qlen {
		return false
	}

	if hplen == 0 && qlen == 0 {
		return true
	}

	for k, v := range hp.keys {
		if qv, ok := o.keys[k]; !ok || v != qv {
			return false
		}
		if hp.data[v].Val != o.data[o.keys[k]].Val {
			return false
		}
	}

	for k, v := range o.keys {
		if hpv, ok := hp.keys[k]; !ok || v != hpv {
			return false
		}
		if o.data[v].Val != hp.data[hp.keys[k]].Val {
			return false
		}
	}

	return true
}

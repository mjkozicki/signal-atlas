package nfc

import (
	"encoding/hex"
	"testing"
)

func TestNDEFTextURIAndMalformed(t *testing.T) {
	cases := []struct{ raw, text, uri string }{{"d101085402656e48656c6c6f", "Hello", ""}, {"d1010c55046578616d706c652e636f6d", "", "https://example.com"}, {"d101095482656efeFF00480069", "Hi", ""}}
	for _, tt := range cases {
		data, e := hex.DecodeString(tt.raw)
		if e != nil {
			t.Fatal(e)
		}
		r, e := ParseNDEF(data)
		if e != nil || len(r) != 1 || r[0].Text != tt.text || r[0].URI != tt.uri {
			t.Fatalf("%s => %+v %v", tt.raw, r, e)
		}
	}
	for _, raw := range []string{"d101", "d101ff54", "9101015500", "d9010054", "f101015400", "d101035405656e", "d1010155ff", "d000"} {
		b, _ := hex.DecodeString(raw)
		if _, e := ParseNDEF(b); e == nil {
			t.Fatalf("malformed %s accepted", raw)
		}
	}
}
func FuzzNDEF(f *testing.F) {
	f.Add([]byte{0xd1, 1, 4, 'T', 2, 'e', 'n', 'A'})
	f.Fuzz(func(t *testing.T, b []byte) { ParseNDEF(b) })
}
func TestNFCDemo(t *testing.T) {
	p := New("python3")
	r := p.Demo()
	if len(r.Devices) != 1 {
		t.Fatal("missing synthetic tag")
	}
}

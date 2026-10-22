package semver

import "testing"

func vs(ss ...string) []Version {
	var out []Version
	for _, s := range ss {
		v, err := Parse(s)
		if err != nil {
			panic(err)
		}
		out = append(out, v)
	}
	return out
}

// CMP-1
func TestParseRanges(t *testing.T) {
	cases := []struct {
		r, v string
		ok   bool
	}{
		{"1.2.3", "1.2.3", true},
		{"1.2.3", "1.2.4", false},
		{"^1.2", "1.9.0", true},
		{"^1.2", "2.0.0", false},
		{"^1.2", "1.1.9", false},
		{"^0.1", "0.1.7", true},
		{"^0.1", "0.2.0", false},
		{"^0.1.2", "0.1.1", false},
		{"head", "1.0.0", false},
		{"^0.1", "head", true},
	}
	for _, c := range cases {
		r, err := ParseRange(c.r)
		if err != nil {
			t.Fatalf("%s: %v", c.r, err)
		}
		v, _ := Parse(c.v)
		if got := r.Allows(v); got != c.ok {
			t.Errorf("%s allows %s = %v, want %v", c.r, c.v, got, c.ok)
		}
	}
	for _, bad := range []string{"1.2", "^x", "01.2.3", "1.2.3-beta", ""} {
		if _, err := ParseRange(bad); err == nil {
			t.Errorf("ParseRange(%q) accepted", bad)
		}
	}
}

func TestLowest(t *testing.T) {
	r, _ := ParseRange("^0.1")
	v, ok := r.Lowest(vs("0.2.0", "0.1.3", "0.1.1", "head"))
	if !ok || v.String() != "0.1.1" {
		t.Fatalf("got %v %v", v, ok)
	}
	h, _ := ParseRange("head")
	if v, ok := h.Lowest(vs("0.1.0", "head")); !ok || !v.Head {
		t.Fatalf("head: got %v %v", v, ok)
	}
	if _, ok := r.Lowest(vs("1.0.0")); ok {
		t.Fatal("expected none")
	}
}

func TestCompare(t *testing.T) {
	v := vs("1.10.0", "1.9.9")
	if Compare(v[0], v[1]) != 1 || Compare(Version{Head: true}, v[0]) != 1 {
		t.Fatal("compare order")
	}
}

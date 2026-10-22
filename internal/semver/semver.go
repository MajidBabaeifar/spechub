// Package semver parses MAJOR.MINOR.PATCH versions and SpecHub ranges (CMP-1).
package semver

import (
	"fmt"
	"strconv"
	"strings"
)

// Version is a release version; Head marks the default-branch head.
type Version struct {
	Major, Minor, Patch int
	Head                bool
}

// Parse accepts "1.2.3" or "v1.2.3", or "head".
func Parse(s string) (Version, error) {
	if s == "head" {
		return Version{Head: true}, nil
	}
	s = strings.TrimPrefix(s, "v")
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return Version{}, fmt.Errorf("version %q: want MAJOR.MINOR.PATCH", s)
	}
	var n [3]int
	for i, p := range parts {
		v, err := strconv.Atoi(p)
		if err != nil || v < 0 || (len(p) > 1 && p[0] == '0') {
			return Version{}, fmt.Errorf("version %q: bad number %q", s, p)
		}
		n[i] = v
	}
	return Version{Major: n[0], Minor: n[1], Patch: n[2]}, nil
}

func (v Version) String() string {
	if v.Head {
		return "head"
	}
	return fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
}

// Compare orders versions; head is greater than every release.
func Compare(a, b Version) int {
	switch {
	case a.Head && b.Head:
		return 0
	case a.Head:
		return 1
	case b.Head:
		return -1
	}
	for _, d := range [3]int{a.Major - b.Major, a.Minor - b.Minor, a.Patch - b.Patch} {
		if d != 0 {
			if d < 0 {
				return -1
			}
			return 1
		}
	}
	return 0
}

// Range is an exact version, a caret range, or head.
type Range struct {
	raw   string
	head  bool
	exact bool
	min   Version
	// caret upper bound, exclusive
	max Version
}

// ParseRange accepts "1.2.3", "^1.2", "^1.2.3", "^1" and "head".
func ParseRange(s string) (Range, error) {
	s = strings.TrimSpace(s)
	if s == "head" {
		return Range{raw: s, head: true}, nil
	}
	if !strings.HasPrefix(s, "^") {
		v, err := Parse(s)
		if err != nil {
			return Range{}, fmt.Errorf("range %q: %w", s, err)
		}
		return Range{raw: s, exact: true, min: v}, nil
	}
	body := strings.TrimPrefix(s, "^")
	parts := strings.Split(body, ".")
	for len(parts) < 3 {
		parts = append(parts, "0")
	}
	v, err := Parse(strings.Join(parts, "."))
	if err != nil {
		return Range{}, fmt.Errorf("range %q: %w", s, err)
	}
	var max Version
	switch {
	case v.Major > 0:
		max = Version{Major: v.Major + 1}
	default:
		max = Version{Minor: v.Minor + 1}
	}
	return Range{raw: s, min: v, max: max}, nil
}

func (r Range) String() string { return r.raw }

// IsHead reports whether the range asks for the default-branch head.
func (r Range) IsHead() bool { return r.head }

// Allows reports whether v satisfies the range. Head satisfies every range.
func (r Range) Allows(v Version) bool {
	if v.Head {
		return true
	}
	if r.head {
		return false
	}
	if r.exact {
		return Compare(v, r.min) == 0
	}
	return Compare(v, r.min) >= 0 && Compare(v, r.max) < 0
}

// Lowest returns the lowest of the given versions the range allows.
func (r Range) Lowest(available []Version) (Version, bool) {
	if r.head {
		for _, v := range available {
			if v.Head {
				return v, true
			}
		}
		return Version{}, false
	}
	var best Version
	found := false
	for _, v := range available {
		if v.Head || !r.Allows(v) {
			continue
		}
		if !found || Compare(v, best) < 0 {
			best, found = v, true
		}
	}
	return best, found
}

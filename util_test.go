package main

import (
	"fmt"
	"testing"
)

func ExamplePrefix() {
	path := "foo/bar/baz"
	for prefix, k := Prefix(path, 0); k != -1; prefix, k = Prefix(path, k) {
		fmt.Printf("%q ", prefix)
	}
	// Output: "" "foo" "foo/bar" "foo/bar/baz"
}

func TestPrefix(t *testing.T) {
	cases := []struct {
		name     string
		path     string
		expected []string
	}{
		{
			name:     lineno(),
			path:     "",
			expected: []string{""},
		},
		{
			name:     lineno(),
			path:     "foo",
			expected: []string{"", "foo"},
		},
		{
			name:     lineno(),
			path:     "foo/bar",
			expected: []string{"", "foo", "foo/bar"},
		},
		{
			name:     lineno(),
			path:     "foo/bar/baz",
			expected: []string{"", "foo", "foo/bar", "foo/bar/baz"},
		},

		// Cases that probably should not happen
		// but which we still want to handle gracefully.
		{
			name:     lineno(),
			path:     "/foo",
			expected: []string{"", "/foo"},
		},
		{
			name:     lineno(),
			path:     "foo/",
			expected: []string{"", "foo", "foo/"},
		},
		{
			name:     lineno(),
			path:     "foo/bar/",
			expected: []string{"", "foo", "foo/bar", "foo/bar/"},
		},
		{
			name:     lineno(),
			path:     "/",
			expected: []string{"", "/"},
		},
		{
			name:     lineno(),
			path:     "//",
			expected: []string{"", "/", "//"},
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			actual := []string{}
			for prefix, k := Prefix(c.path, 0); k != -1; prefix, k = Prefix(c.path, k) {
				actual = append(actual, prefix)
			}
			assertEqual(t, actual, c.expected)
		})
	}
}

func ExampleDeleteEnv() {
	vars := []string{"USER=joe", "UID=1001", "PATH=/bin", "PAGER=less"}
	vars = DeleteEnv(vars, "UID", "HOME", "PAGER")
	fmt.Println(vars)
	// Output: [USER=joe PATH=/bin]
}

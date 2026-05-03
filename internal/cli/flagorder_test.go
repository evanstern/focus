package cli

import (
	"reflect"
	"testing"
)

func TestReorderFlags(t *testing.T) {
	cases := []struct {
		name string
		in   []string
		want []string
	}{
		{
			"flags after positional",
			[]string{"title", "--priority", "p1"},
			[]string{"--priority", "p1", "title"},
		},
		{
			"flags before positional already correct",
			[]string{"--priority", "p1", "title"},
			[]string{"--priority", "p1", "title"},
		},
		{
			"bool flag",
			[]string{"42", "--force"},
			[]string{"--force", "42"},
		},
		{
			"flag=value form",
			[]string{"42", "--priority=p0"},
			[]string{"--priority=p0", "42"},
		},
		{
			"double dash terminator",
			[]string{"--priority", "p1", "--", "--literal-arg"},
			[]string{"--priority", "p1", "--literal-arg"},
		},
		{
			"multi-word title with flag in middle",
			[]string{"Launch", "v2", "--type", "epic"},
			[]string{"--type", "epic", "Launch", "v2"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := reorderFlags(tc.in)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("reorderFlags(%v) = %v, want %v", tc.in, got, tc.want)
			}
		})
	}
}

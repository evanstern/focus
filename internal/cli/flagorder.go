package cli

import "strings"

// reorderFlags moves flag arguments (those starting with "-") to the
// front of args so flag.Parse can see them regardless of where the
// user typed them. Without this, `focus new "title" --priority p1`
// would treat "--priority" as a positional argument because Go's
// flag package stops scanning at the first non-flag token.
//
// Two-token flags ("-flag value") are kept together. "--" terminates
// flag processing the way the conventional Unix flag handlers do.
//
// We intentionally don't try to be clever about whether a flag takes
// a value — we treat any token that follows a flag and doesn't itself
// start with "-" as the value if the flag is one of the registered
// boolean-only flags this CLI uses (--force, --quiet). That's a
// short, explicit list.
func reorderFlags(args []string) []string {
	var flags, positional []string
	boolFlags := map[string]bool{
		"--force": true,
		"-force":  true,
		"--quiet": true,
		"-quiet":  true,
		"-q":      true,
	}
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			positional = append(positional, args[i+1:]...)
			break
		}
		if strings.HasPrefix(a, "-") {
			if strings.Contains(a, "=") || boolFlags[a] {
				flags = append(flags, a)
				continue
			}
			flags = append(flags, a)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				flags = append(flags, args[i+1])
				i++
			}
			continue
		}
		positional = append(positional, a)
	}
	return append(flags, positional...)
}

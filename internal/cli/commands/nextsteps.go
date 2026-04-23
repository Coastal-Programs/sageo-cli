package commands

import (
	"fmt"
	"io"
)

// printNextSteps writes a "Next steps:" block to w. Always targets stderr so
// stdout remains a parseable JSON envelope. The caller supplies the exact
// commands to run, in order. No-op when steps is empty.
//
// The format matches what agents and humans expect from production CLIs
// (gh, flyctl, kubebuilder, cloudquery): a blank leading line, a heading,
// and a numbered list of literal commands.
func printNextSteps(w io.Writer, steps []string) {
	if len(steps) == 0 || w == nil {
		return
	}
	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, "Next steps:")
	for i, s := range steps {
		_, _ = fmt.Fprintf(w, "  %d. %s\n", i+1, s)
	}
}

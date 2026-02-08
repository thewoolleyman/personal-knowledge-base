//go:build acceptance

package acceptance

import "testing"

func TestAcceptBreakCI(t *testing.T) {
	t.Fatal("deliberate acceptance test failure for CI bead verification (pkb-343)")
}

package acceptance

import (
	"os/exec"
	"testing"
)

func Step(moveCmdFlags []string, t *testing.T) {
	dryRun := exec.Command("aztfmove", append(moveCmdFlags, "-auto-approve", "-dry-run")...)
	out, err := dryRun.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", out)
	}
	t.Log("Dry-run succeeded")

	// Test move
	move := exec.Command("aztfmove", append(moveCmdFlags, "-auto-approve")...)
	out, err = move.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", out)
	}
	t.Log("Move succeeded")
}

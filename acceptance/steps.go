package acceptance

import (
	"fmt"
	"os/exec"
	"testing"
)

func Step(moveCmdFlags []string, t *testing.T) {
	dryRun := exec.Command("aztfmove", append(moveCmdFlags, "-auto-approve", "-dry-run", "-no-color")...)
	out, err := dryRun.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", out)
	}
	t.Logf("aztfmove output: \n\n%s", out)
	t.Log("Dry-run succeeded")

	// Test move
	move := exec.Command("aztfmove", append(moveCmdFlags, "-auto-approve", "-no-color")...)
	out, err = move.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", out)
	}
	t.Logf("aztfmove output: \n\n%s", out)
	t.Log("Move succeeded")
}

func StepWithErrorReturn(moveCmdFlags []string, t *testing.T) error {
	dryRun := exec.Command("aztfmove", append(moveCmdFlags, "-auto-approve", "-dry-run", "-no-color")...)
	out, err := dryRun.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aztfmove errorred out: \n\n%s", out)
	}
	t.Logf("aztfmove output: \n\n%s", out)
	t.Log("Dry-run succeeded")

	// Test move
	move := exec.Command("aztfmove", append(moveCmdFlags, "-auto-approve", "-no-color")...)
	out, err = move.CombinedOutput()
	if err != nil {
		return fmt.Errorf("aztfmove errorred out: \n\n%s", out)
	}
	t.Logf("aztfmove output: \n\n%s", out)
	t.Log("Move succeeded")

	return nil
}

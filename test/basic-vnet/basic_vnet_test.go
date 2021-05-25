// +build acctest
// NOTE: We use build tags to differentiate acceptance testing

package test

import (
	"os/exec"
	"testing"

	"github.com/arschles/assert"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestAztfmoveVNet(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "./",
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test dry-run
	cmdDryRun := exec.Command("aztfmove", "-resource=azurerm_virtual_network.vnet", "-target-resource-group=output-rg", "-auto-approve", "-dry-run")
	outDryRun, err := cmdDryRun.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outDryRun)
	}
	t.Log("Dry-run succeeded")

	// Test move
	cmdMove := exec.Command("aztfmove", "-resource=azurerm_virtual_network.vnet", "-target-resource-group=output-rg", "-auto-approve")
	outMove, err := cmdMove.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outMove)
	}
	t.Log("First move succeeded")

	// Test dry-run
	cmdDryRunBack := exec.Command("aztfmove", "-target-resource-group=input-rg", "-auto-approve", "-dry-run")
	outDryRunBack, err := cmdDryRunBack.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outDryRunBack)
	}
	t.Log("Dry-run 2 succeeded")

	// Test move
	cmdMoveBack := exec.Command("aztfmove", "-target-resource-group=input-rg", "-auto-approve")
	outMoveBack, err := cmdMoveBack.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outMoveBack)
	}
	t.Log("Second move succeeded")

	// Validate move
	exitCode := terraform.InitAndPlanWithExitCode(t, terraformOptions)
	assert.Equal(t, exitCode, 0, "Exitcode must be 0")
}

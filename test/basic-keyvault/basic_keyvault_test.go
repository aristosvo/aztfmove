// +build acctest
// NOTE: We use build tags to differentiate acceptance testing

package test

import (
	"fmt"
	"log"
	"os/exec"
	"testing"

	"github.com/arschles/assert"
	"github.com/gruntwork-io/terratest/modules/terraform"

	externalip "github.com/glendc/go-external-ip"
)

func TestAztfmoveKeyVault(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"test": 123,
			"ip":   ipCIDR(),
		},
		VarFiles: []string{"create.tfvars"},
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	// Test dry-run
	cmdDryRun := exec.Command("aztfmove", "-resource-group=input-kv-rg", "-target-resource-group=output-kv-rg", "-auto-approve", "-dry-run")
	outDryRun, err := cmdDryRun.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outDryRun)
	}
	t.Log("Dry-run succeeded")

	// Test move
	cmdMove := exec.Command("aztfmove", "-resource-group=input-kv-rg", "-target-resource-group=output-kv-rg", "-var-file=moved.tfvars", "-var", fmt.Sprintf("ip=%s", ipCIDR()), "-var", "test=123", "-auto-approve")
	outMove, err := cmdMove.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outMove)
	}
	t.Log("First move succeeded")

	// Validate move
	terraformOptionsBack := &terraform.Options{
		TerraformDir: "./",
		Vars: map[string]interface{}{
			"test": 123,
			"ip":   ipCIDR(),
		},
		VarFiles: []string{"moved.tfvars"},
	}
	exitCode := terraform.InitAndPlanWithExitCode(t, terraformOptionsBack)
	assert.Equal(t, exitCode, 0, "Exitcode must be 0")
	t.Log("First move validated")

	// Test dry-run
	cmdDryRunBack := exec.Command("aztfmove", "-resource-group=output-kv-rg", "-target-resource-group=input-kv-rg", "-auto-approve", "-dry-run")
	outDryRunBack, err := cmdDryRunBack.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outDryRunBack)
	}
	t.Log("Dry-run 2 succeeded")

	// Test move
	cmdMoveBack := exec.Command("aztfmove", "-resource-group=output-kv-rg", "-target-resource-group=input-kv-rg", "-var-file=create.tfvars", "-var", fmt.Sprintf("ip=%s", ipCIDR()), "-var", "test=123", "-auto-approve")
	outMoveBack, err := cmdMoveBack.CombinedOutput()
	if err != nil {
		t.Errorf("aztfmove errorred out: \n\n%s", outMoveBack)
	}
	t.Log("Second move succeeded")

	// Validate move
	exitCode = terraform.InitAndPlanWithExitCode(t, terraformOptions)
	assert.Equal(t, exitCode, 0, "Exitcode must be 0")
}

func ipCIDR() string {
	consensus := externalip.DefaultConsensus(nil, nil)
	ip, err := consensus.ExternalIP()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s/32", ip.String())
}

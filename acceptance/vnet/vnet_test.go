// +build acctest
// NOTE: We use build tags to differentiate acceptance testing

package test

import (
	"testing"

	"github.com/aristosvo/aztfmove/acceptance"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestVNet_Basic(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "./",
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	moveVnet := []string{"-resource=azurerm_virtual_network.vnet", "-target-resource-group=output-rg"}
	acceptance.Step(moveVnet, t)

	moveVnetBack := []string{"-target-resource-group=input-rg"}
	acceptance.Step(moveVnetBack, t)

	exitCode := terraform.InitAndPlanWithExitCode(t, terraformOptions)
	if exitCode != 0 {
		t.Fatalf("terraform plan exitcode %d, not %d", exitCode, 0)
	}
}

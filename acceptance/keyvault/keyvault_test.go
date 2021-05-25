// +build acctest
// NOTE: We use build tags to differentiate acceptance testing

package test

import (
	"fmt"
	"log"
	"testing"

	"github.com/aristosvo/aztfmove/acceptance"
	"github.com/arschles/assert"
	"github.com/gruntwork-io/terratest/modules/terraform"

	externalip "github.com/glendc/go-external-ip"
)

func TestKeyVault_Basic(t *testing.T) {
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

	moveKeyVault := []string{"-resource-group=input-kv-rg", "-target-resource-group=output-kv-rg", "-var-file=moved.tfvars", "-var", fmt.Sprintf("ip=%s", ipCIDR()), "-var", "test=123"}
	acceptance.Step(moveKeyVault, t)

	terraformOptions.VarFiles = []string{"moved.tfvars"}
	exitCode := terraform.InitAndPlanWithExitCode(t, terraformOptions)
	assert.Equal(t, exitCode, 0, "terraform plan exitcode")
	t.Log("Move validated")

	moveKeyVaultBack := []string{"-resource-group=output-kv-rg", "-target-resource-group=input-kv-rg", "-var-file=create.tfvars", "-var", fmt.Sprintf("ip=%s", ipCIDR()), "-var", "test=123"}
	acceptance.Step(moveKeyVaultBack, t)

	terraformOptions.VarFiles = []string{"create.tfvars"}
	exitCode = terraform.InitAndPlanWithExitCode(t, terraformOptions)
	assert.Equal(t, exitCode, 0, "terraform plan exitcode")
}

func ipCIDR() string {
	consensus := externalip.DefaultConsensus(nil, nil)
	ip, err := consensus.ExternalIP()
	if err != nil {
		log.Fatal(err)
	}

	return fmt.Sprintf("%s/32", ip.String())
}

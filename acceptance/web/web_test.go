// +build acctest
// NOTE: We use build tags to differentiate acceptance testing

package test

import (
	"testing"

	"github.com/aristosvo/aztfmove/acceptance"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestWeb_Basic(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "./",
		NoColor:      true,
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	moveWeb := []string{"-resource-group=input-web-rg", "-target-resource-group=output-web-rg"}
	acceptance.Step(moveWeb, t)

	moveWebBack := []string{"-target-resource-group=input-web-rg"}
	acceptance.Step(moveWebBack, t)

	terraformOptions = &terraform.Options{
		TerraformDir: "./",
		NoColor:      true,

		// Excluded:
		// - `azurerm_app_service_virtual_network_swift_connection.app_service_virtual_network_swift_connection`
		// - `azurerm_app_service_slot_virtual_network_swift_connection.app_service_slot_virtual_network_swift_connection`
		// These are expected to be recreated after the move
		Targets: []string{
			"azurerm_resource_group.input_rg",
			"azurerm_resource_group.output_rg",
			"random_password.web_postfix",
			"azurerm_monitor_action_group.monitor_action_group",
			"azurerm_virtual_network.vnet",
			"azurerm_subnet.appservice_subnet",
			"azurerm_app_service_plan.app_service_plan",
			"azurerm_monitor_metric_alert.monitor_metric_alert_cpu",
			"azurerm_monitor_metric_alert.monitor_metric_alert_mem",
			"azurerm_app_service.app_service",
			"azurerm_app_service_slot.app_service_slot",
		},
	}

	exitCode := terraform.InitAndPlanWithExitCode(t, terraformOptions)
	if exitCode != 0 {
		t.Fatalf("terraform plan exitcode %d, not %d", exitCode, 0)
	}

	acceptance.Cleanup(t)
}

// +build acctest
// NOTE: We use build tags to differentiate acceptance testing

package test

import (
	"testing"

	"github.com/aristosvo/aztfmove/acceptance"
	"github.com/gruntwork-io/terratest/modules/terraform"
)

func TestMsSql_Basic(t *testing.T) {
	t.Parallel()

	terraformOptions := &terraform.Options{
		TerraformDir: "./",
	}
	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	moveMsSql := []string{"-resource-group=input-sa-rg", "-target-resource-group=output-sa-rg"}
	acceptance.Step(moveMsSql, t)

	moveMsSqlBack := []string{"-target-resource-group=input-sa-rg"}
	acceptance.Step(moveMsSqlBack, t)


	terraformOptions = &terraform.Options{
		TerraformDir: "./",
		// `azurerm_mssql_server.mssql_server` is excluded in the plan, as `administrator_login_password` would be updated. Resolution would be to make use of AAD login without normal administrator enabled
		Targets: []string{
			"azurerm_resource_group.input_rg",
			"azurerm_resource_group.output_rg",
			"azurerm_mssql_database.mssql_db",
			"azurerm_sql_firewall_rule.rule1",
			"azurerm_sql_firewall_rule.rule2",
			"azurerm_mssql_database_extended_auditing_policy.mssql_database_extended_auditing_policy",
			"azurerm_log_analytics_workspace.log_analytics_workspace",
			"azurerm_monitor_diagnostic_setting.diagnostic_setting",
		}
	}
	exitCode := terraform.InitAndPlanWithExitCode(t, terraformOptions)
	if exitCode != 0 {
		t.Fatalf("terraform plan exitcode %d, not %d", exitCode, 0)
	}

	acceptance.Cleanup(t)
}

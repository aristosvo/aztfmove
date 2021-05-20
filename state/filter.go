package state

import (
	"fmt"
	"strings"
)

type ResourceInstanceSummary struct {
	AzureID       string
	TerraformID   string
	FutureAzureID string
	Type          string
}

type ResourcesInstanceSummary []ResourceInstanceSummary

func (ris ResourcesInstanceSummary) NotSupported() []string {
	var IDs []string
	for _, r := range ris {
		if contains(resourcesNotSupportedInAzure, r.Type) {
			IDs = append(IDs, r.TerraformID)
		}
	}
	return IDs
}

func (ris ResourcesInstanceSummary) MovableOnAzure() []string {
	var IDs []string
	for _, r := range ris {
		if !contains(resourcesNotSupportedInAzure, r.Type) && !contains(resourcesOnlyMovedInTF, r.Type) {
			IDs = append(IDs, r.AzureID)
		}
	}
	return IDs
}

func (ris ResourcesInstanceSummary) ToCorrectInTFState() map[string]string {
	IDs := make(map[string]string)
	for _, r := range ris {
		if !contains(resourcesNotSupportedInAzure, r.Type) {
			IDs[r.TerraformID] = r.FutureAzureID
		}
	}
	return IDs
}

var resourcesOnlyMovedInTF = []string{
	"azurerm_mysql_firewall_rule",
	"azurerm_key_vault_access_policy",
	"azurerm_storage_container",
	"azurerm_key_vault_secret",
	"azurerm_storage_share",
}

var resourcesNotSupportedInAzure = []string{
	"azurerm_kubernetes_cluster",
	"azurerm_resource_group",
	"azurerm_client_config",
	"azurerm_monitor_diagnostic_setting",
}

func (tfstate TerraformState) Filter(resourceFilter, moduleFilter, resourceGroupFilter, sourceSubscriptionID, targetResourceGroup, targetSubscriptionID string) (resourceInstances ResourcesInstanceSummary, sourceResourceGroup string, err error) {
	resourceGroup := resourceGroupFilter
	for _, r := range tfstate.Resources {
		if !strings.Contains(r.Provider, "provider[\"registry.terraform.io/hashicorp/azurerm\"]") {
			continue
		}

		// first filter: resource
		if (resourceFilter != "" && resourceFilter != "*") && r.ID() != resourceFilter {
			continue
		}

		// second filter: module
		if (moduleFilter != "" && moduleFilter != "*") && r.Module != moduleFilter {
			continue
		}

		for _, instance := range r.Instances {
			if instance.SubscriptionID() == "" {
				err = fmt.Errorf("subscription ID is not found for %s. Please file a PR on https://github.com/aristosvo/aztfmove and mention this ID: %s", instance.ID(r), instance.ID(r))
				return nil, "", err
			}

			// Only one subscription is supported at the same time
			if instance.SubscriptionID() != sourceSubscriptionID {
				err = fmt.Errorf("resource instance `%s` has a different subscription specified, unable to start moving. Resource instance subscription ID: %s, specified subscription ID: %s", instance.ID(r), strings.Split(instance.Attributes.ID, "/")[2], sourceSubscriptionID)
				return nil, "", err
			}

			instanceResourceGroup := instance.ResourceGroup()
			if instanceResourceGroup == "" && !contains(resourcesNotSupportedInAzure, r.Type) {
				err = fmt.Errorf("resource group is not found for %s. Please file a PR on https://github.com/aristosvo/aztfmove and mention this ID: %s", instance.ID(r), instance.ID(r))
				return nil, "", err
			}

			// thirth filter: resource group
			if resourceGroupFilter != "*" && instanceResourceGroup != resourceGroupFilter {
				continue
			}

			// Only one resource group is supported at the same time
			if resourceGroup == "*" && !contains(resourcesNotSupportedInAzure, r.Type) {
				resourceGroup = instanceResourceGroup
			} else if resourceGroup != instanceResourceGroup && !contains(resourcesNotSupportedInAzure, r.Type) {

				err = fmt.Errorf("multiple resource groups found within your selection, unable to start moving. Resource groups found: [%s, %s]", resourceGroup, instanceResourceGroup)
				return nil, "", err
			}

			if instanceResourceGroup == targetResourceGroup && !contains(resourcesNotSupportedInAzure, r.Type) {
				err = fmt.Errorf("the selected resource %s is already in the target resource group", instance.ID(r))
				return nil, "", err
			}

			// Prepare formatting of ID after movement. Maybe this could be extracted from the movement response?
			// IDs which are formatted like /subscriptions/*/resourceGroups/* are considered sensitive for movement, IDs like https://example.blob.core.windows.net/container not
			resourceGroupId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", sourceSubscriptionID, resourceGroup)
			targetResourceGroupId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", targetSubscriptionID, targetResourceGroup)
			futureAzureId := instance.Attributes.ID
			if strings.HasPrefix(instance.Attributes.ID, resourceGroupId) {
				futureAzureId = strings.Replace(instance.Attributes.ID, resourceGroupId, targetResourceGroupId, 1)
			}

			summary := ResourceInstanceSummary{
				AzureID:       instance.Attributes.ID,
				FutureAzureID: futureAzureId,
				TerraformID:   instance.ID(r),
				Type:          r.Type,
			}
			resourceInstances = append(resourceInstances, summary)
		}
	}
	return resourceInstances, resourceGroup, nil
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

type TerraformState struct {
	Resources []Resource
}

type Resource struct {
	Type      string
	Name      string
	Provider  string
	Module    string
	Instances []Instance
}

type Instance struct {
	IndexKey   interface{} `json:"index_key,omitempty"`
	Attributes Attributes
}

type Attributes struct {
	ID                string
	KeyVaultID        string `json:"key_vault_id,omitempty"`
	ResourceManagerID string `json:"resource_manager_id,omitempty"`
}

type ResourceInstanceSummary struct {
	AzureID       string
	TerraformID   string
	FutureAzureID string
	MoveOnAzure   bool
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
}

func main() {
	var resourceFlag = flag.String("resource", "*", "Terraform resource to be moved.")
	var moduleFlag = flag.String("module", "", "Terraform module to be moved. Optional.")
	var resourceGroupFlag = flag.String("resource-group", "", "Azure resource group to be moved. Optional.")
	var subscriptionFlag = flag.String("subscription-id", "", "subscription where resources are currently. `ARM_SUBSCRIPTION_ID` has the same functionality. Optional.")
	var targetResourceGroupFlag = flag.String("target-resource-group", "", "Azure resource group name where resources are moved. Required.")
	var targetSubscriptionFlag = flag.String("target-subscription-id", "", "Azure subscription ID where resources are moved. If not specified resources are moved within the subscription. Optional.")
	flag.Parse()

	if *targetResourceGroupFlag == "" {
		fmt.Println("[ERROR] `resource` (which can also be a module) and target-resource-group are both required variables")
		os.Exit(1)
	}

	subscriptionId := ""
	if os.Getenv("ARM_SUBSCRIPTION_ID") != "" {
		subscriptionId = os.Getenv("ARM_SUBSCRIPTION_ID")
	} else if *subscriptionFlag != "" {
		subscriptionId = *subscriptionFlag
	} else {
		fmt.Println("[ERROR] No resource subscription known, specify environment variable ARM_SUBSCRIPTION_ID or flag -subscription-id")
		os.Exit(1)
	}

	if *targetSubscriptionFlag == "" {
		fmt.Println("No target subscription specified, move will be within the same subscription:")
		fmt.Printf(" %s -> %s \n", subscriptionId, subscriptionId)
	} else {
		fmt.Println("Target subscription specified, move will be to a different subscription:")
		fmt.Printf(" %s -> %s \n", subscriptionId, *targetSubscriptionFlag)
	}

	cmd := exec.Command("terraform", "state", "pull")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		fmt.Printf("[ERROR] Terraform state is not found. Try `terraform init`.")
		os.Exit(1)
	}

	var data TerraformState
	json.Unmarshal(out.Bytes(), &data)

	fmt.Printf("\nResources selected:\n")
	var resourceInstancesContainer []ResourceInstanceSummary
	resourceGroup := *resourceGroupFlag
	for _, r := range data.Resources {
		if !strings.Contains(r.Provider, "provider[\"registry.terraform.io/hashicorp/azurerm\"]") {
			continue
		}
		modulePrefix := ""
		if r.Module != "" {
			modulePrefix = fmt.Sprintf("%s.", r.Module)
		}
		terraformResourceID := fmt.Sprintf("%s%s.%s", modulePrefix, r.Type, r.Name)

		// first filter: resource
		if (*resourceFlag != "" && *resourceFlag != "*") && terraformResourceID != *resourceFlag {
			continue
		}

		// second filter: module
		if *moduleFlag != "" && r.Module == *moduleFlag {
			continue
		}

		if contains(resourcesNotSupportedInAzure, r.Type) {
			continue
		}

		for _, instance := range r.Instances {
			index := ""
			if instance.IndexKey != nil {
				index = fmt.Sprintf("[\"%v\"]", instance.IndexKey)
			}
			terraformInstanceID := fmt.Sprintf("%s%s", terraformResourceID, index)

			// Only one subscription is supported at the same time
			// attributes `key_vault_id` or `resource_manager_id` are used to find resource group and possibly subscription if ID doesn't contain these
			instanceSubscriptionId := ""
			if strings.HasPrefix(instance.Attributes.ID, "/subscriptions/") {
				instanceSubscriptionId = strings.Split(instance.Attributes.ID, "/")[2]
			} else if instance.Attributes.ResourceManagerID != "" {
				instanceSubscriptionId = strings.Split(instance.Attributes.ResourceManagerID, "/")[2]
			} else if instance.Attributes.KeyVaultID != "" {
				instanceSubscriptionId = strings.Split(instance.Attributes.KeyVaultID, "/")[2]
			} else {
				fmt.Printf("[ERROR] Subscription ID is not found for %s\n", terraformInstanceID)
				fmt.Printf("  Please file a PR on https://github.com/aristosvo/aztfmove and mention this ID: %s\n", terraformInstanceID)

				os.Exit(1)
			}

			if instanceSubscriptionId != subscriptionId {
				fmt.Printf("[ERROR] Resource instance `%s` has a different subscription specified, unable to start moving\n", terraformInstanceID)
				fmt.Printf(" Resource instance subscription ID : %s\n", strings.Split(instance.Attributes.ID, "/")[2])
				fmt.Printf(" Specified subscription ID : %s\n", subscriptionId)

				os.Exit(1)
			}

			// Only one resource group is supported at the same time
			instanceResourceGroupId := ""
			if strings.HasPrefix(instance.Attributes.ID, fmt.Sprintf("/subscriptions/%s/resourceGroups/", subscriptionId)) {
				instanceResourceGroupId = strings.Split(instance.Attributes.ID, "/")[4]
			} else if instance.Attributes.ResourceManagerID != "" {
				instanceResourceGroupId = strings.Split(instance.Attributes.ResourceManagerID, "/")[4]
			} else if instance.Attributes.KeyVaultID != "" {
				instanceResourceGroupId = strings.Split(instance.Attributes.KeyVaultID, "/")[4]
			} else {
				fmt.Printf("[ERROR] Resource group is not found for %s\n", terraformInstanceID)

				os.Exit(1)
			}

			// thirth filter: resource group
			if *resourceGroupFlag != "" && instanceResourceGroupId != *resourceGroupFlag {
				continue
			}

			if resourceGroup == "" {
				resourceGroup = instanceResourceGroupId
			} else if resourceGroup != instanceResourceGroupId {
				fmt.Printf("[ERROR] Multiple resource groups found within your selection, unable to start moving\n")
				fmt.Printf(" Resource groups found : [%s, %s]\n", resourceGroup, instanceResourceGroupId)

				os.Exit(1)
			}

			// Prepare formatting of ID after movement. Maybe this could be extracted from the movement response?
			// IDs which are formatted like /subscriptions/*/resourceGroups/* are considered sensitive for movement, IDs like https://example.blob.core.windows.net/container not
			resourceGroupId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionId, resourceGroup)
			targetResourceGroupId := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionId, *targetResourceGroupFlag)
			futureAzureId := instance.Attributes.ID
			if strings.HasPrefix(instance.Attributes.ID, resourceGroupId) {
				futureAzureId = strings.Replace(instance.Attributes.ID, resourceGroupId, targetResourceGroupId, 1)
			}

			fmt.Println(" -", terraformInstanceID)
			summary := ResourceInstanceSummary{AzureID: instance.Attributes.ID,
				FutureAzureID: futureAzureId,
				TerraformID:   terraformInstanceID,
				MoveOnAzure:   !contains(resourcesOnlyMovedInTF, r.Type)}
			resourceInstancesContainer = append(resourceInstancesContainer, summary)

		}
	}

	fmt.Printf("\nResources moved in Azure:\n")
	var azureIDs []string
	for _, rs := range resourceInstancesContainer {
		if rs.MoveOnAzure {
			azureIDs = append(azureIDs, rs.AzureID)
			fmt.Println(" -", rs.AzureID)
		}
	}

	if len(azureIDs) > 0 {
		targetResourceGroupID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", subscriptionId, *targetResourceGroupFlag)
		moveInfo := resources.MoveInfo{
			ResourcesProperty:   &azureIDs,
			TargetResourceGroup: &targetResourceGroupID,
		}

		resourceClient := resources.NewClient(subscriptionId)
		authorizer, err := auth.NewAuthorizerFromCLI()
		if err == nil {
			resourceClient.Authorizer = authorizer
		}
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
		defer cancel()

		future, err := resourceClient.MoveResources(ctx, resourceGroup, moveInfo)
		if err != nil {
			fmt.Printf("[ERROR] Cannot move resources: %v", err)
			os.Exit(1)
		}
		fmt.Printf("\nResources are to be validated and moved.")
		err = future.WaitForCompletionRef(ctx, resourceClient.Client)
		if err != nil {
			fmt.Printf("[ERROR] cannot get the move future response: %v", err)
			os.Exit(1)
		}
		fmt.Printf("\n\nResources are moved to the specified resource group.")
	}

	fmt.Printf("\n\nResources are removed and imported in Terraform state:\n")
	for _, rs := range resourceInstancesContainer {
		fmt.Println(" -", rs.TerraformID)
		fmt.Printf("  Resource will be removed..\n")
		cmdRemove := exec.Command("terraform", "state", "rm", rs.TerraformID)
		var outRemove bytes.Buffer
		cmdRemove.Stdout = &outRemove
		cmdRemove.Stderr = &outRemove
		err := cmdRemove.Run()
		if err != nil {
			fmt.Println(outRemove.String())
			fmt.Printf("\n[ERROR] Terraform state is not removed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  Resource is removed and will be imported..\n")

		cmdImport := exec.Command("terraform", "import", rs.TerraformID, rs.FutureAzureID)
		var outImport bytes.Buffer
		cmdImport.Stdout = &outImport
		cmdImport.Stderr = &outImport
		err = cmdImport.Run()
		if err != nil {
			fmt.Println(outImport.String())
			fmt.Printf("\n[ERROR] Terraform resource is not imported: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("  Resource is imported..\n")
	}

	fmt.Printf("\nResources are moved and correctly imported in Terraform.\n")
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

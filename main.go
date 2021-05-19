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

func (r Resource) ID() string {
	modulePrefix := ""
	if r.Module != "" {
		modulePrefix = fmt.Sprintf("%s.", r.Module)
	}
	return fmt.Sprintf("%s%s.%s", modulePrefix, r.Type, r.Name)
}

type Instance struct {
	IndexKey   interface{} `json:"index_key,omitempty"`
	Attributes Attributes
}

func (i Instance) ID(r Resource) string {
	index := ""
	if i.IndexKey != nil {
		index = fmt.Sprintf("[\"%v\"]", i.IndexKey)
	}
	return fmt.Sprintf("%s%s", r.ID(), index)
}

func (i Instance) SubscriptionID() string {
	if strings.HasPrefix(i.Attributes.ID, "/subscriptions/") {
		return strings.Split(i.Attributes.ID, "/")[2]
	}

	// attributes `key_vault_id` or `resource_manager_id` are used to find subscription if ID doesn't contain these
	if i.Attributes.ResourceManagerID != "" {
		return strings.Split(i.Attributes.ResourceManagerID, "/")[2]
	}
	if i.Attributes.KeyVaultID != "" {
		return strings.Split(i.Attributes.KeyVaultID, "/")[2]
	}

	return ""
}

func (i Instance) ResourceGroup(subscriptionId string) string {
	if strings.HasPrefix(i.Attributes.ID, fmt.Sprintf("/subscriptions/%s/resourceGroups/", subscriptionId)) {
		return strings.Split(i.Attributes.ID, "/")[4]
	}

	// attributes `key_vault_id` or `resource_manager_id` are used to find resource group if ID doesn't contain these
	if i.Attributes.ResourceManagerID != "" {
		return strings.Split(i.Attributes.ResourceManagerID, "/")[4]
	}
	if i.Attributes.KeyVaultID != "" {
		return strings.Split(i.Attributes.KeyVaultID, "/")[4]
	}

	return ""
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
	NotSupported  bool
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
	var resourceFlag = flag.String("resource", "*", "Terraform resource to be moved. For example \"module.storage.azurerm_storage_account.example\".")
	var moduleFlag = flag.String("module", "*", "Terraform module to be moved. For example \"module.storage\".")
	var resourceGroupFlag = flag.String("resource-group", "*", "Azure resource group to be moved. For example \"example-source-resource-group\".")
	var subscriptionFlag = flag.String("subscription-id", os.Getenv("ARM_SUBSCRIPTION_ID"), "subscription where resources are currently. Environment variable \"ARM_SUBSCRIPTION_ID\" has the same functionality.")
	var targetResourceGroupFlag = flag.String("target-resource-group", "", "Azure resource group name where resources are moved. For example \"example-target-resource-group\". (required)")
	var targetSubscriptionFlag = flag.String("target-subscription-id", os.Getenv("ARM_SUBSCRIPTION_ID"), "Azure subscription ID where resources are moved. If not specified resources are moved within the subscription.")

	// Future functionality:
	// var excludeResourcesFlag = flag.String("exclude-resources", "-", "Terraform resources to be excluded from moving. For example \"module.storage.azurerm_storage_account.example,module.storage.azurerm_storage_account.example\".")
	// var autoApproveFlag = flag.String("auto-approve", "false", "aztfmove first shows which resources are selected for a move both in Azure and in Terraform and requires approval. If you don't want to approve, use this flag.")
	flag.Parse()

	if *targetResourceGroupFlag == "" {
		fmt.Println("[ERROR] `resource` (which can also be a module) and target-resource-group are both required variables")
		os.Exit(1)
	}

	subscriptionId := ""
	if *subscriptionFlag != "" {
		subscriptionId = *subscriptionFlag
	} else {
		fmt.Println("[ERROR] No resource subscription known, specify environment variable ARM_SUBSCRIPTION_ID or flag -subscription-id")
		os.Exit(1)
	}

	if *targetSubscriptionFlag == "" || *targetSubscriptionFlag == *subscriptionFlag {
		fmt.Println("No unique \"-target-subscription-id\" specified, move will be within the same subscription:")
		fmt.Printf(" %s -> %s \n", subscriptionId, subscriptionId)
	} else {
		fmt.Println("Target subscription specified, move will be to a different subscription:")
		fmt.Printf(" %s -> %s \n", subscriptionId, *targetSubscriptionFlag)
	}

	tfstate, err := pullTerraformState()
	if err != nil {
		fmt.Printf("[ERROR] Terraform state is not found. Try `terraform init`.")
		os.Exit(1)
	}

	var resourceInstances []ResourceInstanceSummary
	resourceGroup := *resourceGroupFlag
	for _, r := range tfstate.Resources {
		if !strings.Contains(r.Provider, "provider[\"registry.terraform.io/hashicorp/azurerm\"]") {
			continue
		}

		// first filter: resource
		if (*resourceFlag != "" && *resourceFlag != "*") && r.ID() != *resourceFlag {
			continue
		}

		// second filter: module
		if (*moduleFlag != "" && *moduleFlag != "*") && r.Module == *moduleFlag {
			continue
		}

		for _, instance := range r.Instances {
			if instance.SubscriptionID() == "" {
				fmt.Printf("[ERROR] Subscription ID is not found for %s\n", instance.ID(r))
				fmt.Printf("  Please file a PR on https://github.com/aristosvo/aztfmove and mention this ID: %s\n", instance.ID(r))
				os.Exit(1)
			}

			// Only one subscription is supported at the same time
			if instance.SubscriptionID() != subscriptionId {
				fmt.Printf("[ERROR] Resource instance `%s` has a different subscription specified, unable to start moving\n", instance.ID(r))
				fmt.Printf(" Resource instance subscription ID : %s\n", strings.Split(instance.Attributes.ID, "/")[2])
				fmt.Printf(" Specified subscription ID : %s\n", subscriptionId)
				os.Exit(1)
			}

			instanceResourceGroup := instance.ResourceGroup(subscriptionId)
			if instanceResourceGroup == "" {
				fmt.Printf("[ERROR] Resource group is not found for %s\n", instance.ID(r))
				fmt.Printf("  Please file a PR on https://github.com/aristosvo/aztfmove and mention this ID: %s\n", instance.ID(r))
				os.Exit(1)
			}

			// thirth filter: resource group
			if *resourceGroupFlag != "*" && instanceResourceGroup != *resourceGroupFlag {
				continue
			}

			// Only one resource group is supported at the same time
			if resourceGroup == "*" {
				resourceGroup = instanceResourceGroup
			} else if resourceGroup != instanceResourceGroup {
				fmt.Printf("[ERROR] Multiple resource groups found within your selection, unable to start moving\n")
				fmt.Printf(" Resource groups found : [%s, %s]\n", resourceGroup, instanceResourceGroup)

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

			summary := ResourceInstanceSummary{
				AzureID:       instance.Attributes.ID,
				FutureAzureID: futureAzureId,
				TerraformID:   instance.ID(r),
				MoveOnAzure:   !contains(resourcesOnlyMovedInTF, r.Type),
				NotSupported:  contains(resourcesNotSupportedInAzure, r.Type),
			}
			resourceInstances = append(resourceInstances, summary)
		}
	}

	fmt.Printf("\nResources not supported for movement:\n")
	for _, rs := range resourceInstances {
		if rs.NotSupported {
			fmt.Println(" -", rs.TerraformID)
		}
	}

	fmt.Printf("\nResources moved in Terraform:\n")
	for _, rs := range resourceInstances {
		if !rs.NotSupported {
			fmt.Println(" -", rs.TerraformID)
		}
	}

	fmt.Printf("\nResources moved in Azure:\n")
	var azureIDs []string
	for _, rs := range resourceInstances {
		if rs.MoveOnAzure && !rs.NotSupported {
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
	for _, rs := range resourceInstances {
		fmt.Println(" -", rs.TerraformID)

		fmt.Printf("    Resource will be removed..\n")
		output, err := rs.removeFromTFState()
		if err != nil {
			fmt.Println(output)
			fmt.Printf("\n[ERROR] Terraform state is not removed: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("    Resource is removed and will be imported..\n")

		output, err = rs.importInTFState()
		if err != nil {
			fmt.Println(output)
			fmt.Printf("\n[ERROR] Terraform resource is not imported: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("    Resource is imported..\n")
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

func (ris ResourceInstanceSummary) removeFromTFState() (string, error) {
	cmdRemove := exec.Command("terraform", "state", "rm", ris.TerraformID)
	var outRemove bytes.Buffer
	cmdRemove.Stdout = &outRemove
	cmdRemove.Stderr = &outRemove

	return outRemove.String(), cmdRemove.Run()
}

func (ris ResourceInstanceSummary) importInTFState() (string, error) {
	cmdImport := exec.Command("terraform", "import", ris.TerraformID, ris.FutureAzureID)
	var outImport bytes.Buffer
	cmdImport.Stdout = &outImport
	cmdImport.Stderr = &outImport

	return outImport.String(), cmdImport.Run()
}

func pullTerraformState() (TerraformState, error) {
	var tfstate TerraformState
	cmd := exec.Command("terraform", "state", "pull")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return tfstate, err
	}
	json.Unmarshal(out.Bytes(), &tfstate)
	return tfstate, nil
}

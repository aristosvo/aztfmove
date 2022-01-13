package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2020-10-01/resources"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/aristosvo/aztfmove/state"
)

var (
	tfVars     state.ArrayVars
	tfVarFiles state.ArrayVarFiles

	// TODO: should probably refactor `-resource` and `-module` to `-target` to mimic terraform flags as much as possible
	resourceFlag            = flag.String("resource", "*", "Terraform resource to be moved. For example 'module.storage.azurerm_storage_account.example'.")
	moduleFlag              = flag.String("module", "*", "Terraform module to be moved. For example 'module.storage'.")
	sourceResourceGroupFlag = flag.String("resource-group", "*", "Azure resource group to be moved. For example 'example-source-resource-group'.")
	sourceSubscriptionFlag  = flag.String("subscription-id", os.Getenv("ARM_SUBSCRIPTION_ID"), "subscription where resources are currently. Environment variable 'ARM_SUBSCRIPTION_ID' has the same functionality.")
	targetResourceGroupFlag = flag.String("target-resource-group", "", "Azure resource group name where resources are moved. For example 'example-target-resource-group'. (required)")
	targetSubscriptionFlag  = flag.String("target-subscription-id", *sourceSubscriptionFlag, "Azure subscription ID where resources are moved. If not specified resources are moved within the subscription.")
	autoApproveFlag         = flag.Bool("auto-approve", false, "aztfmove first shows which resources are selected for a move and requires approval. If you want to approve automatically, use this flag.")
	dryRunFlag              = flag.Bool("dry-run", false, "if set to true, aztfmove only shows which resources are selected for a move.")
	noColorFlag             = flag.Bool("no-color", false, "if set to true, aztfmove prints without color.")
	// TODO: var excludeResourcesFlag = flag.String("exclude-resources", "-", "Terraform resources to be excluded from moving. For example 'module.storage.azurerm_storage_account.example,module.storage.azurerm_storage_account.example'.")
	// but..., this is not according to previously stated principle to mimic terraform flags as much as possible
)

func init() {
	flag.Var(&tfVars, "var", "use this like you'd use Terraform \"-var\", i.e. \"-var 'test1=123' -var 'test2=312'\" ")
	flag.Var(&tfVarFiles, "var-file", "use this like you'd use Terraform \"-var-file\", i.e. \"-var-file='tst.tfvars'\" ")
}

var (
	Azure        = TealBold
	AzureCLI     = Teal
	Warn         = Yellow
	Fata         = Red
	Good         = Green
	Terraform    = CyanBold
	TerraformCLI = Cyan
)

var (
	Red      = Color("\033[1;31m%s\033[0m")
	Green    = Color("\033[1;32m%s\033[0m")
	Yellow   = Color("\033[1;33m%s\033[0m")
	Teal     = Color("\033[36m%s\033[0m")
	TealBold = Color("\033[1;36m%s\033[0m")
	CyanBold = Color("\033[1;35m%s\033[0m")
	Cyan     = Color("\033[35m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		if *noColorFlag && strings.Contains(colorString, "1;") {
			return fmt.Sprintf("\033[1m%s\033[0m",
				fmt.Sprint(args...))
		} else if *noColorFlag {
			return fmt.Sprint(args...)
		}
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

func main() {
	flag.Parse()
	validateInput()

	if *targetSubscriptionFlag == "" || *targetSubscriptionFlag == *sourceSubscriptionFlag {
		fmt.Println(Good("No unique \"-target-subscription-id\" specified, move will be within the same subscription:"))
		*targetSubscriptionFlag = *sourceSubscriptionFlag
	} else {
		fmt.Println(Good("Target subscription specified, move will be to a different subscription:"))
	}
	fmt.Printf(" %s -> %s \n", *sourceSubscriptionFlag, *targetSubscriptionFlag)

	tfstate, err := state.PullRemote()
	if err != nil {
		fmt.Printf("%s Terraform state is not found. Try `terraform init`.", Fata("Error:"))
		os.Exit(1)
	}

	resourceInstances, sourceResourceGroup, err := tfstate.Filter(*resourceFlag, *moduleFlag, *sourceResourceGroupFlag, *sourceSubscriptionFlag, *targetResourceGroupFlag, *targetSubscriptionFlag)
	if err != nil {
		fmt.Printf("%s %v", Fata("Error:"), err)
		os.Exit(1)
	}

	printBlockingMovement(resourceInstances.BlockingMovement())
	printNotSupported(resourceInstances.NotSupported())
	printNotNeeded(resourceInstances.NoMovementNeeded())
	printToMoveInAzure(resourceInstances.MovableOnAzure())
	printToCorrectInTF(resourceInstances.ToCorrectInTFState())

	if !*dryRunFlag && !*autoApproveFlag {
		askConfirmation()
	}

	if tfIDsToRemove, azureIDsToDelete := resourceInstances.BlockingMovement(); len(azureIDsToDelete) > 0 {
		fmt.Print(Azure("\nBlocking resources will be deleted in Azure."))
		if *dryRunFlag {
			fmt.Print(" (dry-run!)")
		}
		deleteAzureResources(azureIDsToDelete, sourceResourceGroup, *sourceSubscriptionFlag)
		fmt.Print(Good("\n\nBlocking resources are deleted in Azure."))
		if *dryRunFlag {
			fmt.Print(" (dry-run!)")
		}
		fmt.Print(Terraform("\n\nResources in Terraform state will be removed:"))
		if *dryRunFlag {
			fmt.Print(" (dry-run!)")
		}
		removeTerraformResources(tfIDsToRemove)
	}

	if azureIDs := resourceInstances.MovableOnAzure(); len(azureIDs) > 0 {
		fmt.Print(Azure("\nResources are on the move to the specified resource group."))
		if *dryRunFlag {
			fmt.Print(" (dry-run!)")
		}
		moveAzureResources(azureIDs, sourceResourceGroup, *sourceSubscriptionFlag, *targetSubscriptionFlag, *targetResourceGroupFlag)
		fmt.Print(Good("\n\nResources are moved to the specified resource group."))
		if *dryRunFlag {
			fmt.Print(" (dry-run!)")
		}
	}

	fmt.Print(Terraform("\n\nResources in Terraform state are enhanced:"))
	if *dryRunFlag {
		fmt.Print(" (dry-run!)")
	}
	reimportTerraformResources(resourceInstances.ToCorrectInTFState())

	if *dryRunFlag {
		fmt.Print(Good("\nDry-run complete!\n"))
		fmt.Printf("Resources are not moved to the specified resource group, but the resources actions (and corresponding %s and %s commands) are visible above.\n", Azure("az cli"), Terraform("terraform"))
		os.Exit(0)
	}

	fmt.Print(Good("\n\nCongratulations! Resources are moved in Azure and corrected in Terraform.\n"))
}

func validateInput() {
	if *targetResourceGroupFlag == "" {
		fmt.Printf("%s target-resource-group is a required variables\n", Fata("Error:"))
		os.Exit(1)
	}

	if *sourceSubscriptionFlag == "" {
		fmt.Printf("%s No resource subscription known, specify environment variable ARM_SUBSCRIPTION_ID or flag -subscription-id\n", Fata("Error:"))
		os.Exit(1)
	}
}

func printBlockingMovement(terraformIDs []string, azureIDs []string) {
	if len(terraformIDs) == 0 {
		return
	}
	fmt.Print(Warn("\nResources blocking movement of other resources:\n"))
	for _, id := range terraformIDs {
		fmt.Println(" -", id)
	}
}

func printNotSupported(terraformIDs []string) {
	if len(terraformIDs) == 0 {
		return
	}
	fmt.Print(Warn("\nResources not supported for movement:\n"))
	for _, id := range terraformIDs {
		fmt.Println(" -", id)
	}
}

func printNotNeeded(terraformIDs []string) {
	if len(terraformIDs) == 0 {
		return
	}
	fmt.Print(Good("\nResources with no need for movement:\n (mostly child resources)\n"))
	for _, id := range terraformIDs {
		fmt.Println(" -", id)
	}
}

func printToCorrectInTF(resources map[string]string) {
	fmt.Print(Terraform("\nResources to be corrected in Terraform:\n"))
	for k, v := range resources {
		fmt.Printf(" - %s: [id=%s]\n", k, v)
	}
}

func printToMoveInAzure(azureIDs []string) {
	fmt.Print(Azure("\nResources to be moved in Azure:\n"))
	for _, id := range azureIDs {
		fmt.Println(" -", id)
	}
}

func askConfirmation() {
	fmt.Print(Good("\nCan you confirm these resources should be moved?"))
	if *dryRunFlag {
		fmt.Print(" (dry-run!)")
	}
	fmt.Printf("\nCheck the Azure documentation on moving Azure resources (https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/move-resource-group-and-subscription) for all the details for your specific resources.")
	fmt.Print(Good("\n\nType 'yes' to confirm: "))

	reader := bufio.NewReader(os.Stdin)
	inputString, err := reader.ReadString('\n')
	if err != nil {
		fmt.Printf("%s confirmation of the import errored out\n", Fata("Error:"))
		os.Exit(1)
	}
	inputString = strings.TrimSuffix(inputString, "\n")
	if inputString != "yes" {
		fmt.Printf("\nMove is canceled\n")
		os.Exit(0)

	}
}

func deleteAzureResources(azureIDs []string, sourceResourceGroup string, sourceSubscriptionID string) {
	if *dryRunFlag {
		fmt.Println("\nThe Azure delete actions when \"-dry-run=false\" are similar to the scripted action below:")
		fmt.Printf(AzureCLI("  az resource delete --ids '%s'"), strings.Join(azureIDs, " "))

		return
	}

	resourceClient := resources.NewClient(sourceSubscriptionID)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		resourceClient.Authorizer = authorizer
	}
	for _, id := range azureIDs {
		ctx, cancel := context.WithTimeout(context.Background(), 60*60*time.Second)
		defer cancel()

		// TODO: API Version hardcoded as only one category yet implemented
		future, err := resourceClient.DeleteByID(ctx, id, "2021-02-01")
		if err != nil {
			fmt.Printf("\n%s cannot delete resources: %v", Fata("Error:"), err)
			os.Exit(1)
		}
		err = future.WaitForCompletionRef(ctx, resourceClient.Client)
		if err != nil {
			fmt.Printf("\n%s cannot get the delete future response: %v", Fata("Error:"), err)
			os.Exit(1)
		}
	}

}

func moveAzureResources(azureIDs []string, sourceResourceGroup string, sourceSubscriptionID string, targetSubscriptionID string, targetResourceGroup string) {
	if *dryRunFlag {
		fmt.Println("\nThe Azure move actions when \"-dry-run=false\" are similar to the scripted action below:")
		fmt.Printf(AzureCLI("  az resource move --destination-group '%s' --destination-subscription-id '%s' --ids '%s'"), targetResourceGroup, targetSubscriptionID, strings.Join(azureIDs, " "))

		return
	}

	fmt.Printf("\nIt can take some time before this is done, don't panic!")
	targetResourceGroupID := fmt.Sprintf("/subscriptions/%s/resourceGroups/%s", targetSubscriptionID, targetResourceGroup)
	moveInfo := resources.MoveInfo{
		ResourcesProperty:   &azureIDs,
		TargetResourceGroup: &targetResourceGroupID,
	}

	resourceClient := resources.NewClient(sourceSubscriptionID)
	authorizer, err := auth.NewAuthorizerFromCLI()
	if err == nil {
		resourceClient.Authorizer = authorizer
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*60*time.Second)
	defer cancel()

	future, err := resourceClient.MoveResources(ctx, sourceResourceGroup, moveInfo)
	if err != nil {
		fmt.Printf("\n%s cannot move resources: %v", Fata("Error:"), err)
		os.Exit(1)
	}
	err = future.WaitForCompletionRef(ctx, resourceClient.Client)
	if err != nil {
		fmt.Printf("\n%s cannot get the move future response: %v", Fata("Error:"), err)
		os.Exit(1)
	}
}

func removeTerraformResources(tfIDs []string) {
	if *dryRunFlag {
		fmt.Println("\nThe Terraform actions taken when \"-dry-run=false\" are similar to the scripted actions below:")
		for _, tfID := range tfIDs {
			fmt.Println(" #", tfID)
			fmt.Printf(TerraformCLI("  terraform state rm '%s'\n"), tfID)
		}
		return
	}

	for _, tfID := range tfIDs {
		fmt.Println("\n -", tfID)

		output, err := state.RemoveInstance(tfID)
		if err != nil {
			fmt.Printf("\n%s terraform resource is not removed, %v\n", Fata("Error:"), err)
			fmt.Println(" ", output)
			os.Exit(1)
		}
		fmt.Printf("\t✓ Removed")
	}
}

func reimportTerraformResources(resources map[string]string) {
	if *dryRunFlag {
		fmt.Println("\nThe Terraform actions taken when \"-dry-run=false\" are similar to the scripted actions below:")
		for tfID, newAzureID := range resources {
			fmt.Println(" #", tfID)
			fmt.Printf(TerraformCLI("  terraform state rm '%s'\n"), tfID)
			fmt.Printf(TerraformCLI("  terraform import %s %s '%s' '%s'\n"), strings.Join(tfVarFiles, " "), strings.Join(tfVars, " "), tfID, newAzureID)
		}
		return
	}

	for tfID, newAzureID := range resources {
		fmt.Println("\n -", tfID)

		output, err := state.RemoveInstance(tfID)
		if err != nil {
			fmt.Printf("\n%s terraform resource is not removed, %v\n", Fata("Error:"), err)
			fmt.Println(" ", output)
			os.Exit(1)
		}
		fmt.Printf("\t✓ Removed")

		output, err = state.ImportInstance(tfID, newAzureID, tfVars, tfVarFiles)
		if err != nil {
			fmt.Printf("\n%s terraform resource is not imported, %v\n", Fata("Error:"), err)
			fmt.Println(" ", output)
			os.Exit(1)
		}
		fmt.Printf("\t✓ Imported")
	}
}

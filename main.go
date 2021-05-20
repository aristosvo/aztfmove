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
	Azure = Teal
	Warn  = Yellow
	Fata  = Red
	Good  = Green
)

var (
	Red    = Color("\033[1;31m%s\033[0m")
	Green  = Color("\033[1;32m%s\033[0m")
	Yellow = Color("\033[1;33m%s\033[0m")
	Teal   = Color("\033[1;36m%s\033[0m")
)

func Color(colorString string) func(...interface{}) string {
	sprint := func(args ...interface{}) string {
		return fmt.Sprintf(colorString,
			fmt.Sprint(args...))
	}
	return sprint
}

var (
	tfVars     state.ArrayVars
	tfVarFiles state.ArrayVarFiles
)

func main() {
	// TODO: should probably refactor `-resource` and `-module` to `-target` to mimic terraform flags as much as possible
	resourceFlag := flag.String("resource", "*", "Terraform resource to be moved. For example \"module.storage.azurerm_storage_account.example\".")
	moduleFlag := flag.String("module", "*", "Terraform module to be moved. For example \"module.storage\".")
	resourceGroupFlag := flag.String("resource-group", "*", "Azure resource group to be moved. For example \"example-source-resource-group\".")
	subscriptionFlag := flag.String("subscription-id", os.Getenv("ARM_SUBSCRIPTION_ID"), "subscription where resources are currently. Environment variable \"ARM_SUBSCRIPTION_ID\" has the same functionality.")
	targetResourceGroupFlag := flag.String("target-resource-group", "", "Azure resource group name where resources are moved. For example \"example-target-resource-group\". (required)")
	targetSubscriptionFlag := flag.String("target-subscription-id", *subscriptionFlag, "Azure subscription ID where resources are moved. If not specified resources are moved within the subscription.")
	autoApproveFlag := flag.Bool("auto-approve", false, "aztfmove first shows which resources are selected for a move and requires approval. If you want to approve automatically, use this flag.")
	dryRunFlag := flag.Bool("dry-run", false, "if set to true, aztfmove only shows which resources are selected for a move.")
	flag.Var(&tfVars, "var", "use this like you'd use Terraform \"-var\", i.e. \"-var 'test1=123' -var 'test2=312'\" ")
	flag.Var(&tfVarFiles, "var-file", "use this like you'd use Terraform \"-var-file\", i.e. \"-var-file=tst.tfvars\" ")
	// TODO: var excludeResourcesFlag = flag.String("exclude-resources", "-", "Terraform resources to be excluded from moving. For example \"module.storage.azurerm_storage_account.example,module.storage.azurerm_storage_account.example\".")
	// but..., this is not according to previously stated principle to mimic terraform flags as much as possible
	flag.Parse()

	if *targetResourceGroupFlag == "" {
		fmt.Printf("%s target-resource-group is a required variables\n", Fata("Error:"))
		os.Exit(1)
	}

	sourceSubscriptionId := ""
	if *subscriptionFlag != "" {
		sourceSubscriptionId = *subscriptionFlag
	} else {
		fmt.Printf("%s No resource subscription known, specify environment variable ARM_SUBSCRIPTION_ID or flag -subscription-id\n", Fata("Error:"))
		os.Exit(1)
	}

	if *targetSubscriptionFlag == "" || *targetSubscriptionFlag == *subscriptionFlag {
		fmt.Println("No unique \"-target-subscription-id\" specified, move will be within the same subscription:")
		fmt.Printf(" %s -> %s \n", sourceSubscriptionId, sourceSubscriptionId)
		*targetSubscriptionFlag = sourceSubscriptionId
	} else {
		fmt.Println("Target subscription specified, move will be to a different subscription:")
		fmt.Printf(" %s -> %s \n", sourceSubscriptionId, *targetSubscriptionFlag)
	}

	tfstate, err := state.PullRemote()
	if err != nil {
		fmt.Printf("%s Terraform state is not found. Try `terraform init`.", Fata("Error:"))
		os.Exit(1)
	}

	resourceInstances, sourceResourceGroup, err := tfstate.Filter(*resourceFlag, *moduleFlag, *resourceGroupFlag, sourceSubscriptionId, *targetResourceGroupFlag, *targetSubscriptionFlag)
	if err != nil {
		fmt.Printf("%s %v", Fata("Error:"), err)
		os.Exit(1)
	}

	printNotSupported(resourceInstances.NotSupported())
	printToMoveInAzure(resourceInstances.MovableOnAzure())
	printToCorrectInTF(resourceInstances.ToCorrectInTFState())

	if *dryRunFlag {
		fmt.Print(Green("\nDry-run complete!\n"))
		fmt.Print("Resources are not moved to the specified resource group, but the resources which would be moved are visible above.\n")
		os.Exit(0)
	}
	if !*autoApproveFlag {
		askConfirmation()
	}

	if azureIDs := resourceInstances.MovableOnAzure(); len(azureIDs) > 0 {
		fmt.Print(Green("\nResources are on the move to the specified resource group."))
		fmt.Printf("\nIt can take some time before this is done, don't panic!")

		moveAzureResources(azureIDs, sourceResourceGroup, sourceSubscriptionId, *targetSubscriptionFlag, *targetResourceGroupFlag)
		fmt.Print(Green("\n\nResources are moved to the specified resource group."))
	}

	fmt.Printf("\n\nResources in Terraform state are enhanced:\n")
	correctTerraformResources(resourceInstances.ToCorrectInTFState())

	fmt.Print(Green("\nCongratulations! Resources are moved in Azure and corrected in Terraform.\n"))
}

func printNotSupported(terraformIDs []string) {
	fmt.Print(Warn("\nResources not supported for movement:\n"))
	for _, id := range terraformIDs {
		fmt.Println(" -", id)
	}
}

func printToCorrectInTF(resources map[string]string) {
	fmt.Print(Good("\nResources to be corrected in Terraform:\n"))
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
	fmt.Print(Warn("\nCan you confirm these resources should be moved?"))
	fmt.Printf("\nCheck the Azure documentation on moving Azure resources (https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/move-resource-group-and-subscription) for all the details for your specific resources.")
	fmt.Print(Warn("\n\nType 'yes' to confirm: "))
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

func moveAzureResources(azureIDs []string, sourceResourceGroup string, sourceSubscriptionID string, targetSubscriptionID string, targetResourceGroup string) {
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
	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
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

func correctTerraformResources(resources map[string]string) {
	for tfID, newAzureID := range resources {
		fmt.Println(" -", tfID)

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
		fmt.Printf("\t✓ Imported\n")
	}
}

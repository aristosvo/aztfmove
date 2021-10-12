# aztfmove
Simple tool to move Azure resources based on Terraform state

## Goal
It is sometimes inevitable to move Azure resources to a new subscription or to a different resource group. This means often a painfull transition when all resources are already in Terraform.

To combine both the joy of Terraform and the movement capabilities within Azure, this tool gives you the capabilities to migrate quick and easy.

## Usage
```
❯ aztfmove -h
Usage of aztfmove:
  -auto-approve
        aztfmove first shows which resources are selected for a move and requires approval. If you want to approve automatically, use this flag.
  -dry-run
        if set to true, aztfmove only shows which resources are selected for a move.
  -module string
        Terraform module to be moved. For example "module.storage". (default "*")
  -resource string
        Terraform resource to be moved. For example "module.storage.azurerm_storage_account.example". (default "*")
  -resource-group string
        Azure resource group to be moved. For example "example-source-resource-group". (default "*")
  -subscription-id string
        subscription where resources are currently. Environment variable "ARM_SUBSCRIPTION_ID" has the same functionality. (default "3xampl32-uu1d-11eb-8529-0242ac130003")
  -target-resource-group string
        Azure resource group name where resources are moved. For example "example-target-resource-group". (required)
  -target-subscription-id string
        Azure subscription ID where resources are moved. If not specified resources are moved within the subscription. (default "3xampl32-uu1d-11eb-8529-0242ac130003")
  -var value
        use this like you'd use Terraform "-var", i.e. "-var 'test1=123' -var 'test2=312'" 
  -var-file value
        use this like you'd use Terraform "-var-file", i.e. "-var-file=tst.tfvars"
```

## Setup

Run:
```bash
go install github.com/aristosvo/aztfmove
```

## Authentication

Authentication for Terraform is the same for `aztfmove` as for normal `terraform` operations. Start with `terraform init` to make sure the (remote) terraform state is available.

Authentication for the movements in Azure is based on an default [azure-sdk-for-go](https://github.com/Azure/azure-sdk-for-go) authorizer which uses Azure CLI to obtain its credentials.

To use this way of authentication, follow these steps:

1. Install Azure CLI v2.0.12 or later. Upgrade earlier versions.
2. Use az login to sign in to Azure.

If you receive an error, use `az account get-access-token` to verify access.

If Azure CLI is not installed to the default directory, you may receive an error reporting that az cannot be found.
Use the AzureCLIPath environment variable to define the Azure CLI installation folder.

If you are signed in to Azure CLI using multiple accounts or your account has access to multiple subscriptions, you need to specify the specific subscription to be used. To do so, use:

```
az account set --subscription <subscription-id>
```
To verify the current account settings, use:
```
az account list
```

## Examples
For examples and/or tests, see [test](https://github.com/aristosvo/aztfmove/tree/main/test) directory.

Sample output for part of the `basic-storage` test run below:
```
❯ aztfmove -target-resource-group input-sa-rg
No unique "-target-subscription-id" specified, move will be within the same subscription:
 3xampl32-uu1d-11eb-8529-0242ac130003 -> 3xampl32-uu1d-11eb-8529-0242ac130003

Resources not supported for movement:
 - azurerm_resource_group.input-rg
 - azurerm_resource_group.output-rg

Resources to be moved in Azure:
 - /subscriptions/3xampl32-uu1d-11eb-8529-0242ac130003/resourceGroups/output-sa-rg/providers/Microsoft.Storage/storageAccounts/samove9ywva4a1

Resources to be corrected in Terraform:
 - azurerm_storage_account.sa-move
 - azurerm_storage_container.sc-move

Can you confirm these resources should be moved?
Check the Azure documentation to move Azure resources (https://docs.microsoft.com/en-us/azure/azure-resource-manager/management/move-resource-group-and-subscription) for all the details on your specific resources.

Type 'yes' to confirm: yes

Resources are on the move to the specified resource group.
It can take some time before this is done, don't panic!

Resources are moved to the specified resource group.

Resources in Terraform state are enhanced:
 - azurerm_storage_account.sa-move
    ✓ Removed   ✓ Imported
 - azurerm_storage_container.sc-move
    ✓ Removed   ✓ Imported

Congratulations! Resources are moved in Azure and corrected in Terraform.
```


## ToDo
- [ ] Use [terraform-exec](https://github.com/hashicorp/terraform-exec) instead of wrapping `terraform`
- [ ] Multiple authentication options (ideally all options supported in the provider)

## Licence

MIT

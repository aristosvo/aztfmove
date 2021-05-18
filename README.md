# aztfmove
Simple tool to move Azure resources based on Terraform state

## Goal
It is sometimes inevitable to move Azure resources to a new subscription or to a different resource group. This means often a painfull transition when all resources are already in Terraform.

To combine both the joy of Terraform and the movement capabilities within Azure, this tool gives you the capabilities to migrate quick and easy.

## Usage
```
❯ aztfmove -h
Usage of aztfmove:
  -module string
        Terraform module to be moved. Optional.
  -resource *
        Terraform resource to be moved. (default "*")
  -resource-group string
        Azure resource group to be moved. Optional.
  -subscription-id ARM_SUBSCRIPTION_ID
        subscription where resources are currently. ARM_SUBSCRIPTION_ID has the same functionality. Optional.
  -target-resource-group string
        Azure resource group name where resources are moved. Required.
  -target-subscription-id string
        Azure subscription ID where resources are moved. If not specified resources are moved within the subscription. Optional.
```

## Examples
For examples and/or tests, see [test](https://github.com/aristosvo/aztfmove/tree/main/test) directory.

## ToDo
- [ ] Use [terratest](https://terratest.gruntwork.io) or similar for AccTests
- [ ] Rework the code in multiple packages instead of one file
- [ ] Unit tests

#!/bin/bash
set -e

terraform init
terraform apply -auto-approve
aztfmove -resource azurerm_virtual_network.vnet -target-resource-group output-rg -dry-run
aztfmove -resource azurerm_virtual_network.vnet -target-resource-group output-rg -auto-approve
aztfmove -target-resource-group input-rg -dry-run
aztfmove -target-resource-group input-rg -auto-approve
terraform plan -detailed-exitcode
terraform destroy -auto-approve
rm terraform.tfstate*
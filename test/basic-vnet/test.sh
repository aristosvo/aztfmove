#!/bin/bash
set -e

terraform init
terraform apply -auto-approve
aztfmove -resource azurerm_virtual_network.vnet -target-resource-group output-rg
terraform destroy -auto-approve
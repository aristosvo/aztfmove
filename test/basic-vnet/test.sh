#!/bin/bash
set -e

terraform init
terraform apply -auto-approve
echo ""
echo "-------------------------------"
echo "           Stage 1             "
echo " Terraform resources created!  "
echo "-------------------------------"
echo ""
aztfmove -resource azurerm_virtual_network.vnet -target-resource-group output-rg -auto-approve -dry-run
aztfmove -resource azurerm_virtual_network.vnet -target-resource-group output-rg -auto-approve
echo ""
echo "-------------------------------"
echo "          Stage 2              "
echo "  Terraform resources moved!   "
echo "-------------------------------"
echo ""
aztfmove -target-resource-group input-rg -auto-approve -dry-run
aztfmove -target-resource-group input-rg -auto-approve
terraform plan -detailed-exitcode
echo ""
echo "-------------------------------"
echo "          Stage 3              "
echo "Terraform resources moved back!"
echo "-------------------------------"
echo ""
terraform destroy -auto-approve
rm terraform.tfstate*
echo ""
echo "-------------------------------"
echo "          Stage 4              "
echo " Terraform resources destroyed!"
echo "-------------------------------"
echo ""
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
aztfmove -resource-group input-sa-rg -target-resource-group output-sa-rg -auto-approve -dry-run
aztfmove -resource-group input-sa-rg -target-resource-group output-sa-rg -auto-approve
echo ""
echo "-------------------------------"
echo "          Stage 2              "
echo "  Terraform resources moved!   "
echo "-------------------------------"
echo ""
aztfmove -target-resource-group input-sa-rg -auto-approve -dry-run
aztfmove -target-resource-group input-sa-rg -auto-approve
echo ""
echo "-------------------------------"
echo "          Stage 3              "
echo "Terraform resources moved back!"
echo "-------------------------------"
echo ""
terraform plan -detailed-exitcode
terraform destroy -auto-approve
rm terraform.tfstate*
echo ""
echo "-------------------------------"
echo "          Stage 4              "
echo " Terraform resources destroyed!"
echo "-------------------------------"
echo ""
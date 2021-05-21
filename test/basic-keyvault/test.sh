#!/bin/bash
set -e

terraform init
terraform apply -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=create.tfvars -auto-approve
echo ""
echo "-------------------------------"
echo "           Stage 1             "
echo " Terraform resources created!  "
echo "-------------------------------"
echo ""
aztfmove -resource-group input-kv-rg -target-resource-group output-kv-rg -auto-approve -dry-run
aztfmove -resource-group input-kv-rg -target-resource-group output-kv-rg -auto-approve -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=moved.tfvars
echo ""
echo "-------------------------------"
echo "          Stage 2              "
echo "  Terraform resources moved!   "
echo "-------------------------------"
echo ""
aztfmove -target-resource-group input-kv-rg -auto-approve -dry-run -var-file=create.tfvars
aztfmove -target-resource-group input-kv-rg -auto-approve -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=create.tfvars 
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
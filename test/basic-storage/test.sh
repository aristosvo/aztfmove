#!/bin/bash
set -e

terraform init
terraform apply -auto-approve
aztfmove -resource-group input-sa-rg -target-resource-group output-sa-rg -dry-run
aztfmove -resource-group input-sa-rg -target-resource-group output-sa-rg -auto-approve
aztfmove -target-resource-group input-sa-rg -dry-run
aztfmove -target-resource-group input-sa-rg -auto-approve
terraform plan -detailed-exitcode
terraform destroy -auto-approve
rm terraform.tfstate*
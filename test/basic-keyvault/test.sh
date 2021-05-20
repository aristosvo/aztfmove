#!/bin/bash
set -e

terraform init
terraform apply -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=create.tfvars -auto-approve 
aztfmove -resource-group input-kv-rg -target-resource-group output-kv-rg -dry-run
aztfmove -resource-group input-kv-rg -target-resource-group output-kv-rg -auto-approve -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=moved.tfvars
terraform plan -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=moved.tfvars -detailed-exitcode 
aztfmove -target-resource-group input-kv-rg -dry-run -var-file=create.tfvars
aztfmove -target-resource-group input-kv-rg -auto-approve -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=create.tfvars
terraform plan -var ip=$(curl ipinfo.io/ip) -var test=123 -var-file=create.tfvars -detailed-exitcode 
terraform destroy -auto-approve
rm terraform.tfstate*
#!/bin/bash
set -e

terraform init
terraform apply -auto-approve
aztfmove -resource-group input-sa-rg -target-resource-group output-sa-rg
aztfmove -target-resource-group input-sa-rg
terraform destroy -auto-approve
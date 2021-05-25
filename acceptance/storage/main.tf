provider "azurerm" {
  features {}
}

variable "location" {
  default     = "westeurope"
  description = "Locatie for all resources, standard is westeurope or \"West Europe\"."
  validation {
    condition     = can(regex("^westeurope|northeurope$", var.location))
    error_message = "We only use region West Europe and North Europe for now."
  }
}

variable "tags" {
  description = "Tags for all resources"
  type = object({
    Customer    = string
    Team        = string
    Environment = string
  })
  default = {
    Customer    = "test"
    Team        = "aristosvo"
    Environment = "acceptance"
  }
  validation {
    condition     = contains(["test", "staging", "development", "acceptance", "production"], lookup(var.tags, "Environment", "wrong"))
    error_message = "Environment should be one of the values \"test\", \"staging\", \"development\", \"acceptance\" or \"production\"."
  }
}

resource "random_password" "sa-postfix" {
  length  = 8
  special = false
}

resource "azurerm_resource_group" "input-rg" {
  name     = "input-sa-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_resource_group" "output-rg" {
  name     = "output-sa-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_storage_account" "sa-move" {
  name                     = "samove${lower(nonsensitive(random_password.sa-postfix.result))}"
  resource_group_name      = azurerm_resource_group.input-rg.name
  location                 = azurerm_resource_group.input-rg.location
  account_kind             = "StorageV2"
  account_tier             = "Standard"
  account_replication_type = "LRS"

  tags = var.tags

  lifecycle {
    ignore_changes = [secondary_blob_connection_string, secondary_location]
  }
}

resource "azurerm_storage_container" "sc-move" {
  name                  = "scmove"
  storage_account_name  = azurerm_storage_account.sa-move.name
  container_access_type = "private"
}
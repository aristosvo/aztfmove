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

resource "azurerm_resource_group" "input-rg" {
  name     = "input-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_resource_group" "output-rg" {
  name     = "output-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_virtual_network" "vnet" {
  count = 1
  
  name                = "moved-vnet"
  address_space       = ["10.0.0.0/16"]
  resource_group_name = azurerm_resource_group.input-rg.name
  location            = azurerm_resource_group.input-rg.location
  tags                = var.tags
}
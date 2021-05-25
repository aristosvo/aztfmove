provider "azurerm" {
  features {
    key_vault {
      purge_soft_delete_on_destroy = false
    }
  }
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

variable "ip" {
  description = "IP address used to access the Key Vault"
  type        = string
}

variable "test" {
  description = "Extra variable to test variables"
  type        = string
}

variable "stage" {
  description = "Extra variable to test variable files combined with `terraform import`"
  type        = string
  validation {
    condition     = contains(["create", "moved"], var.stage)
    error_message = "Stage should be one of the values \"create\" (to mimic creation state) or  \"moved\" (to mimic moved state)."
  }
}

data "azurerm_client_config" "current" {}

resource "random_password" "kv_pwd" {
  length  = 8
  special = false
}

resource "azurerm_resource_group" "input_rg" {
  name     = "input-kv-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_resource_group" "output_rg" {
  name     = "output-kv-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_key_vault" "kv_move" {
  name                       = "move-kv-${nonsensitive(random_password.kv_pwd.result)}"
  location                   = var.location
  resource_group_name        = var.stage == "create" ? azurerm_resource_group.input_rg.name : azurerm_resource_group.output_rg.name
  tenant_id                  = data.azurerm_client_config.current.tenant_id
  sku_name                   = "standard"
  purge_protection_enabled   = true
  soft_delete_retention_days = 90

  network_acls {
    default_action             = "Deny"
    bypass                     = "None"
    ip_rules                   = [var.ip]
    virtual_network_subnet_ids = []
  }

  tags = var.tags

  lifecycle {
    ignore_changes = [access_policy]
  }
}

resource "azurerm_key_vault_access_policy" "move" {
  key_vault_id            = azurerm_key_vault.kv_move.id
  tenant_id               = data.azurerm_client_config.current.tenant_id
  object_id               = data.azurerm_client_config.current.object_id
  secret_permissions      = ["Delete", "Get", "List", "Set"]
  certificate_permissions = []
  key_permissions         = []
  storage_permissions     = []

}

resource "azurerm_key_vault_secret" "move" {
  depends_on = [azurerm_key_vault_access_policy.move]

  name         = "secret-sauce"
  content_type = ""
  value        = nonsensitive(random_password.kv_pwd.result)
  key_vault_id = azurerm_key_vault.kv_move.id

  lifecycle {
    ignore_changes = [content_type]
  }

  tags = var.tags
}
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

resource "azurerm_resource_group" "input_rg" {
  name     = "input-sa-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_resource_group" "output_rg" {
  name     = "output-sa-rg"
  location = var.location
  tags     = var.tags
}

resource "random_password" "mssql_postfix" {
  length  = 8
  special = false
}

resource "azurerm_mssql_server" "mssql_server" {
  name                         = "sqlsrvr-move-${lower(nonsensitive(random_password.mssql_postfix.result))}"
  resource_group_name          = azurerm_resource_group.input_rg.name
  location                     = azurerm_resource_group.input_rg.location
  version                      = "12.0"
  administrator_login          = "aztfmoveadmin"
  administrator_login_password = "Id0n7kn0wwha$$od0h3re"
  minimum_tls_version          = "1.2"
  tags                         = var.tags
}

resource "azurerm_mssql_database" "mssql_db" {
  name        = "sqldb-move-${lower(nonsensitive(random_password.mssql_postfix.result))}"
  server_id   = azurerm_mssql_server.mssql_server.id
  tags        = var.tags
  max_size_gb = 5
  sku_name    = "S3"
}

resource "azurerm_mssql_database_extended_auditing_policy" "mssql_database_extended_auditing_policy" {
  database_id            = azurerm_mssql_database.mssql_db.id
  log_monitoring_enabled = true
}

resource "azurerm_sql_firewall_rule" "rule1" {
  name                = "one"
  resource_group_name = azurerm_resource_group.input_rg.name
  server_name         = azurerm_mssql_server.mssql_server.name
  start_ip_address    = "8.8.8.8"
  end_ip_address      = "8.8.8.8"
}

resource "azurerm_sql_firewall_rule" "rule2" {
  name                = "two"
  resource_group_name = azurerm_resource_group.input_rg.name
  server_name         = azurerm_mssql_server.mssql_server.name
  start_ip_address    = "9.9.9.9"
  end_ip_address      = "9.9.9.9"
}

resource "azurerm_log_analytics_workspace" "log_analytics_workspace" {
  name                = "law-move-${lower(nonsensitive(random_password.mssql_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  location            = azurerm_resource_group.input_rg.location
  sku                 = "PerGB2018"
  tags                = var.tags
  retention_in_days   = 30
}

resource "azurerm_monitor_diagnostic_setting" "diagnostic_setting" {
  log_analytics_workspace_id = azurerm_log_analytics_workspace.log_analytics_workspace.id
  name                       = "diagnostic-setting-move"
  target_resource_id         = azurerm_mssql_database.mssql_db.id

  log {
    category = "AutomaticTuning"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "Blocks"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "DatabaseWaitStatistics"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "Deadlocks"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "Errors"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "QueryStoreRuntimeStatistics"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "QueryStoreWaitStatistics"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "SQLInsights"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "SQLSecurityAuditEvents"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "Timeouts"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  log {
    category = "DevOpsOperationsAudit"
    enabled  = false

    retention_policy {
      days    = 0
      enabled = false
    }
  }

  metric {
    category = "Basic"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  metric {
    category = "InstanceAndAppAdvanced"
    enabled  = true

    retention_policy {
      days    = 0
      enabled = false
    }
  }
  metric {
    category = "WorkloadManagement"
    enabled  = false

    retention_policy {
      days    = 0
      enabled = false
    }
  }
}

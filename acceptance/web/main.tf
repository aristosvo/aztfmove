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
  name     = "input-web-rg"
  location = var.location
  tags     = var.tags
}

resource "azurerm_resource_group" "output_rg" {
  name     = "output-web-rg"
  location = var.location
  tags     = var.tags
}

resource "random_password" "web_postfix" {
  length  = 8
  special = false
}

resource "azurerm_monitor_action_group" "monitor_action_group" {
  name                = "action-group-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  short_name          = "move"

  email_receiver {
    name                    = "aztfmove"
    email_address           = "test@test.com"
    use_common_alert_schema = true
  }

  tags = var.tags
}

resource "azurerm_virtual_network" "vnet" {
  name                = "vnet-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  location            = azurerm_resource_group.input_rg.location
  address_space       = ["10.1.0.0/16"]

  tags = var.tags
}

resource "azurerm_subnet" "appservice_subnet" {
  name                 = "snet-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name  = azurerm_resource_group.input_rg.name
  virtual_network_name = azurerm_virtual_network.vnet.name
  address_prefixes     = ["10.1.2.0/24"]

  delegation {
    name = "delegation"

    service_delegation {
      name    = "Microsoft.Web/serverFarms"
      actions = ["Microsoft.Network/virtualNetworks/subnets/join/action", "Microsoft.Network/virtualNetworks/subnets/prepareNetworkPolicies/action"]
    }
  }
  service_endpoints = [
    "Microsoft.Sql",
    "Microsoft.Storage",
    "Microsoft.KeyVault",
  ]

  lifecycle {
    ignore_changes = [
      delegation.0.service_delegation
    ]
  }
}

resource "azurerm_app_service_plan" "app_service_plan" {
  name                = "appsp-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  location            = azurerm_resource_group.input_rg.location
  kind                = "app"
  tags                = var.tags

  sku {
    capacity = 1
    tier     = "Standard"
    size     = "S2"
  }
}

resource "azurerm_monitor_metric_alert" "monitor_metric_alert_cpu" {
  name                = "cpu-alert-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  enabled             = true
  scopes              = [azurerm_app_service_plan.app_service_plan.id]

  criteria {
    metric_namespace = "Microsoft.Web/serverFarms"
    metric_name      = "CpuPercentage"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 80
  }

  action {
    action_group_id = azurerm_monitor_action_group.monitor_action_group.id
  }

  tags = var.tags
}

resource "azurerm_monitor_metric_alert" "monitor_metric_alert_mem" {
  name                = "mem-alert-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  enabled             = true
  scopes              = [azurerm_app_service_plan.app_service_plan.id]

  criteria {
    metric_namespace = "Microsoft.Web/serverFarms"
    metric_name      = "MemoryPercentage"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 80
  }

  action {
    action_group_id = azurerm_monitor_action_group.monitor_action_group.id
  }

  tags = var.tags
}

resource "azurerm_app_service" "app_service" {
  name                = "appsvc-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  location            = azurerm_resource_group.input_rg.location
  app_service_plan_id = azurerm_app_service_plan.app_service_plan.id
  https_only          = true
  tags                = var.tags

  site_config {
    vnet_route_all_enabled = true
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "azurerm_app_service_virtual_network_swift_connection" "app_service_virtual_network_swift_connection" {
  app_service_id = azurerm_app_service.app_service.id
  subnet_id      = azurerm_subnet.appservice_subnet.id
}

resource "azurerm_app_service_slot" "app_service_slot" {
  name                = "appsvcslot-move-${lower(nonsensitive(random_password.web_postfix.result))}"
  resource_group_name = azurerm_resource_group.input_rg.name
  location            = azurerm_resource_group.input_rg.location
  app_service_plan_id = azurerm_app_service_plan.app_service_plan.id
  app_service_name    = azurerm_app_service.app_service.name
  https_only          = true
  tags                = var.tags

  site_config {
    vnet_route_all_enabled = true
  }

  identity {
    type = "SystemAssigned"
  }
}

resource "azurerm_app_service_slot_virtual_network_swift_connection" "app_service_slot_virtual_network_swift_connection" {
  slot_name      = azurerm_app_service_slot.app_service_slot.name
  app_service_id = azurerm_app_service.app_service.id
  subnet_id      = azurerm_subnet.appservice_subnet.id
}

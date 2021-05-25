package state

import (
	"fmt"
	"reflect"
	"testing"
)

func TestFilter(t *testing.T) {
	state := TerraformState{
		Resources: []Resource{
			{
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Module:   "",
				Type:     "azurerm_storage_account",
				Name:     "example_storage_1",
				Mode:     "managed",
				Instances: []Instance{
					{
						IndexKey: nil,
						Attributes: Attributes{
							ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup1/providers/Microsoft.Storage/storageAccounts/storageaccount1",
						},
					},
				},
			},
			{
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Module:   "module.storage",
				Type:     "azurerm_storage_account",
				Name:     "example_storage_2",
				Mode:     "managed",
				Instances: []Instance{
					{
						IndexKey: nil,
						Attributes: Attributes{
							ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.Storage/storageAccounts/storageaccount2",
						},
					},
				},
			},
			{
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Module:   "module.test",
				Type:     "azurerm_storage_container",
				Name:     "example_container_1",
				Mode:     "managed",
				Instances: []Instance{
					{
						IndexKey: nil,
						Attributes: Attributes{
							ID:                "https://example.blob.core.windows.net/container_1",
							ResourceManagerID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup1/providers/Microsoft.Storage/storageAccounts/storageaccount1/blobServices/default/containers/container_1",
						},
					},
				},
			},
			{
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Module:   "module.storage",
				Type:     "azurerm_storage_container",
				Name:     "example_container_2",
				Mode:     "managed",
				Instances: []Instance{
					{
						IndexKey: nil,
						Attributes: Attributes{
							ID:                "https://example.blob.core.windows.net/container_2",
							ResourceManagerID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.Storage/storageAccounts/storageaccount2/blobServices/default/containers/container_2",
						},
					},
				},
			},

			{
				Provider: "provider[\"registry.terraform.io/hashicorp/azurerm\"]",
				Module:   "module.test",
				Type:     "azurerm_resource_group",
				Name:     "rg3",
				Mode:     "managed",
				Instances: []Instance{
					{
						IndexKey: nil,
						Attributes: Attributes{
							ID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg3",
						},
					},
				},
			},
		},
	}

	t.Run("No filter", func(t *testing.T) {
		_, _, gotError := state.Filter("*", "*", "*", "00000000-0000-0000-0000-000000000000", "myresourcegroup2", "00000000-0000-0000-0000-000000000001")
		wantedError := fmt.Errorf("multiple resource groups found within your selection, unable to start moving. Resource groups found: [myresourcegroup1, myresourcegroup]")
		if gotError.Error() != wantedError.Error() {
			t.Errorf("got %v wanted %v", gotError, wantedError)
		}
	})

	t.Run("Resource filter", func(t *testing.T) {
		gotSummary, _, _ := state.Filter("module.storage.azurerm_storage_container.example_container_2", "*", "*", "00000000-0000-0000-0000-000000000000", "myresourcegroup2", "00000000-0000-0000-0000-000000000001")
		wantedSummary := ResourcesInstanceSummary{
			{
				AzureID:       "https://example.blob.core.windows.net/container_2",
				TerraformID:   "module.storage.azurerm_storage_container.example_container_2",
				FutureAzureID: "https://example.blob.core.windows.net/container_2",
				Type:          "azurerm_storage_container",
			},
		}
		if !reflect.DeepEqual(gotSummary, wantedSummary) {
			t.Errorf("got %v wanted %v", gotSummary, wantedSummary)
		}
	})

	t.Run("Module filter", func(t *testing.T) {
		gotSummary, _, _ := state.Filter("*", "module.storage", "*", "00000000-0000-0000-0000-000000000000", "myresourcegroup2", "00000000-0000-0000-0000-000000000001")
		wantedSummary := ResourcesInstanceSummary{
			{
				AzureID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.Storage/storageAccounts/storageaccount2",
				TerraformID:   "module.storage.azurerm_storage_account.example_storage_2",
				FutureAzureID: "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/myresourcegroup2/providers/Microsoft.Storage/storageAccounts/storageaccount2",
				Type:          "azurerm_storage_account",
			},
			{
				AzureID:       "https://example.blob.core.windows.net/container_2",
				TerraformID:   "module.storage.azurerm_storage_container.example_container_2",
				FutureAzureID: "https://example.blob.core.windows.net/container_2",
				Type:          "azurerm_storage_container",
			},
		}
		if !reflect.DeepEqual(gotSummary, wantedSummary) {
			t.Errorf("got %v wanted %v", gotSummary, wantedSummary)
		}
	})

	t.Run("Module filter with diff resource group passing", func(t *testing.T) {
		gotSummary, _, _ := state.Filter("*", "module.test", "*", "00000000-0000-0000-0000-000000000000", "myresourcegroup2", "00000000-0000-0000-0000-000000000001")
		wantedSummary := ResourcesInstanceSummary{
			{
				AzureID:       "https://example.blob.core.windows.net/container_1",
				TerraformID:   "module.test.azurerm_storage_container.example_container_1",
				FutureAzureID: "https://example.blob.core.windows.net/container_1",
				Type:          "azurerm_storage_container",
			},
			{
				AzureID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg3",
				TerraformID:   "module.test.azurerm_resource_group.rg3",
				FutureAzureID: "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg3",
				Type:          "azurerm_resource_group",
			},
		}
		if !reflect.DeepEqual(gotSummary, wantedSummary) {
			t.Errorf("got %v wanted %v", gotSummary, wantedSummary)
		}
	})

	t.Run("Resource Group filter", func(t *testing.T) {
		gotSummary, _, _ := state.Filter("*", "*", "rg3", "00000000-0000-0000-0000-000000000000", "myresourcegroup2", "00000000-0000-0000-0000-000000000001")
		wantedSummary := ResourcesInstanceSummary{
			{
				AzureID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg3",
				TerraformID:   "module.test.azurerm_resource_group.rg3",
				FutureAzureID: "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/myresourcegroup2",
				Type:          "azurerm_resource_group",
			},
		}
		if !reflect.DeepEqual(gotSummary, wantedSummary) {
			t.Errorf("got %v wanted %v", gotSummary, wantedSummary)
		}
	})
}

func TestNotSupported(t *testing.T) {
	summary := ResourcesInstanceSummary{
		{
			AzureID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/rg3",
			TerraformID:   "module.test.azurerm_resource_group.rg3",
			FutureAzureID: "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/myresourcegroup2",
			Type:          "azurerm_resource_group",
		},
		{
			AzureID:       "https://example.blob.core.windows.net/container_1",
			TerraformID:   "module.test.azurerm_storage_container.example_container_1",
			FutureAzureID: "https://example.blob.core.windows.net/container_1",
			Type:          "azurerm_storage_container",
		},
		{
			AzureID:       "/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.Storage/storageAccounts/storageaccount2",
			TerraformID:   "module.storage.azurerm_storage_account.example_storage_2",
			FutureAzureID: "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/myresourcegroup2/providers/Microsoft.Storage/storageAccounts/storageaccount2",
			Type:          "azurerm_storage_account",
		},
		{
			AzureID:       "https://example.blob.core.windows.net/container_2",
			TerraformID:   "module.storage.azurerm_storage_container.example_container_2",
			FutureAzureID: "https://example.blob.core.windows.net/container_2",
			Type:          "azurerm_storage_container",
		},
	}
	t.Run("Not Supported list", func(t *testing.T) {
		got := summary.NotSupported()
		wanted := []string{"module.test.azurerm_resource_group.rg3"}

		if !reflect.DeepEqual(got, wanted) {
			t.Errorf("got %v wanted %v", got, wanted)
		}
	})

	t.Run("Movable on Azure list", func(t *testing.T) {
		got := summary.MovableOnAzure()
		wanted := []string{"/subscriptions/00000000-0000-0000-0000-000000000000/resourceGroups/myresourcegroup/providers/Microsoft.Storage/storageAccounts/storageaccount2"}

		if !reflect.DeepEqual(got, wanted) {
			t.Errorf("got %v wanted %v", got, wanted)
		}
	})

	t.Run("To correct in TF map", func(t *testing.T) {
		got := summary.ToCorrectInTFState()
		wanted := map[string]string{
			"module.test.azurerm_storage_container.example_container_1":    "https://example.blob.core.windows.net/container_1",
			"module.storage.azurerm_storage_account.example_storage_2":     "/subscriptions/00000000-0000-0000-0000-000000000001/resourceGroups/myresourcegroup2/providers/Microsoft.Storage/storageAccounts/storageaccount2",
			"module.storage.azurerm_storage_container.example_container_2": "https://example.blob.core.windows.net/container_2",
		}

		if !reflect.DeepEqual(got, wanted) {
			t.Errorf("got %v wanted %v", got, wanted)
		}
	})
}

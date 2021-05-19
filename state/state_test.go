package state

import "testing"

func TestResourceID(t *testing.T) {
	t.Run("With module", func(t *testing.T) {
		resource := Resource{
			Module: "module.storage",
			Type:   "azurerm_storage_account",
			Name:   "example_storage",
		}
		got := resource.ID()
		wanted := "module.storage.azurerm_storage_account.example_storage"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("No module", func(t *testing.T) {
		resource := Resource{
			Module: "",
			Type:   "azurerm_storage_account",
			Name:   "example_storage",
		}
		got := resource.ID()
		wanted := "azurerm_storage_account.example_storage"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})
}
func TestInstanceID(t *testing.T) {
	t.Run("No index", func(t *testing.T) {
		resource := Resource{
			Module: "",
			Type:   "azurerm_storage_account",
			Name:   "example_storage",
			Instances: []Instance{
				{
					IndexKey: nil,
					Attributes: Attributes{
						ID: "",
					},
				},
			},
		}

		got := resource.Instances[0].ID(resource)
		wanted := "azurerm_storage_account.example_storage"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Index as string", func(t *testing.T) {
		resource := Resource{
			Module: "",
			Type:   "azurerm_storage_account",
			Name:   "example_storage",
			Instances: []Instance{
				{
					IndexKey: "testindex",
					Attributes: Attributes{
						ID: "",
					},
				},
			},
		}

		got := resource.Instances[0].ID(resource)
		wanted := "azurerm_storage_account.example_storage[\"testindex\"]"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Index as number", func(t *testing.T) {
		resource := Resource{
			Module: "",
			Type:   "azurerm_storage_account",
			Name:   "example_storage",
			Instances: []Instance{
				Instance{
					IndexKey: 1,
					Attributes: Attributes{
						ID: "",
					},
				},
			},
		}

		got := resource.Instances[0].ID(resource)
		wanted := "azurerm_storage_account.example_storage[\"1\"]"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})
}

func TestInstanceSubscriptionID(t *testing.T) {
	t.Run("Working ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID: "/subscriptions/test/resourceGroups/test123",
			},
		}

		got := instance.SubscriptionID()
		wanted := "test"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Wrong ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID: "/subscription/test/resourceGroups/test123",
			},
		}

		got := instance.SubscriptionID()
		wanted := ""
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Invalid ID working RM ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID:                "/subscription/test/resourceGroups/test123",
				ResourceManagerID: "/subscriptions/test/resourceGroups/test123",
			},
		}

		got := instance.SubscriptionID()
		wanted := "test"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Invalid ID working KV ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID:         "/subscription/test/resourceGroups/test123",
				KeyVaultID: "/subscriptions/test7/resourceGroups/test123",
			},
		}

		got := instance.SubscriptionID()
		wanted := "test7"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Invalid ID non working KV ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID:         "/subscription/test/resourceGroups/test123",
				KeyVaultID: "/subscriptio/test7/resourceGroups/test123",
			},
		}

		got := instance.SubscriptionID()
		wanted := ""
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})
}

func TestInstanceResourceGroup(t *testing.T) {
	t.Run("Working ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID: "/subscriptions/test/resourceGroups/test123",
			},
		}

		got := instance.ResourceGroup()
		wanted := "test123"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Wrong ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID: "/subscription/test/resourceGroups/test123",
			},
		}

		got := instance.ResourceGroup()
		wanted := ""
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Invalid ID working RM ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID:                "/subscription/test/resourceGroups/test123",
				ResourceManagerID: "/subscriptions/test/resourceGroups/test312",
			},
		}

		got := instance.ResourceGroup()
		wanted := "test312"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Invalid ID working KV ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID:         "/subscription/test/resourceGroups/test123",
				KeyVaultID: "/subscriptions/test7/resourceGroups/test312",
			},
		}

		got := instance.ResourceGroup()
		wanted := "test312"
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})

	t.Run("Invalid ID non working KV ID", func(t *testing.T) {
		instance := Instance{
			Attributes: Attributes{
				ID:         "/subscription/test/resourceGroups/test123",
				KeyVaultID: "/subscriptio/test7/resourceGroups/test312",
			},
		}

		got := instance.ResourceGroup()
		wanted := ""
		if got != wanted {
			t.Errorf("got %s wanted %s", got, wanted)
		}
	})
}

package state

import (
	"fmt"
	"strings"
)

type TerraformState struct {
	Resources []Resource
}

type Resource struct {
	Type      string
	Name      string
	Provider  string
	Module    string
	Mode      string
	Instances []Instance
}

func (r Resource) ID() string {
	modulePrefix := ""
	if r.Module != "" {
		modulePrefix = fmt.Sprintf("%s.", r.Module)
	}
	return fmt.Sprintf("%s%s.%s", modulePrefix, r.Type, r.Name)
}

type Instance struct {
	IndexKey   interface{} `json:"index_key,omitempty"`
	Attributes Attributes
}

func (i Instance) ID(r Resource) string {
	index := ""
	if i.IndexKey != nil {
		index = fmt.Sprintf("[\"%v\"]", i.IndexKey)
	}
	return fmt.Sprintf("%s%s", r.ID(), index)
}

func (i Instance) SubscriptionID() string {
	if strings.HasPrefix(i.Attributes.ID, "/subscriptions/") {
		return strings.Split(i.Attributes.ID, "/")[2]
	}

	// attributes `key_vault_id`, `resource_manager_id` or `subscription_id` are used to find subscription if ID doesn't contain these
	if i.Attributes.ResourceManagerID != "" && strings.HasPrefix(i.Attributes.ResourceManagerID, "/subscriptions/") {
		return strings.Split(i.Attributes.ResourceManagerID, "/")[2]
	}
	if i.Attributes.KeyVaultID != "" && strings.HasPrefix(i.Attributes.KeyVaultID, "/subscriptions/") {
		return strings.Split(i.Attributes.KeyVaultID, "/")[2]
	}
	if i.Attributes.SubscriptionID != "" {
		return i.Attributes.SubscriptionID
	}

	return ""
}

func (i Instance) ResourceGroup() string {
	if strings.HasPrefix(i.Attributes.ID, fmt.Sprintf("/subscriptions/%s/resourceGroups/", i.SubscriptionID())) {
		return strings.Split(i.Attributes.ID, "/")[4]
	}

	// attributes `key_vault_id` or `resource_manager_id` are used to find resource group if ID doesn't contain these
	if i.Attributes.ResourceManagerID != "" && strings.HasPrefix(i.Attributes.ResourceManagerID, fmt.Sprintf("/subscriptions/%s/resourceGroups/", i.SubscriptionID())) {
		return strings.Split(i.Attributes.ResourceManagerID, "/")[4]
	}
	if i.Attributes.KeyVaultID != "" && strings.HasPrefix(i.Attributes.KeyVaultID, fmt.Sprintf("/subscriptions/%s/resourceGroups/", i.SubscriptionID())) {
		return strings.Split(i.Attributes.KeyVaultID, "/")[4]
	}

	return ""
}

type Attributes struct {
	ID                string
	KeyVaultID        string `json:"key_vault_id,omitempty"`
	ResourceManagerID string `json:"resource_manager_id,omitempty"`
	SubscriptionID    string `json:"subscription_id,omitempty"`
}

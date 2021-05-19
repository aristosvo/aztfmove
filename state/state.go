package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
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

	// attributes `key_vault_id` or `resource_manager_id` are used to find subscription if ID doesn't contain these
	if i.Attributes.ResourceManagerID != "" && strings.HasPrefix(i.Attributes.ResourceManagerID, "/subscriptions/") {
		return strings.Split(i.Attributes.ResourceManagerID, "/")[2]
	}
	if i.Attributes.KeyVaultID != "" && strings.HasPrefix(i.Attributes.KeyVaultID, "/subscriptions/") {
		return strings.Split(i.Attributes.KeyVaultID, "/")[2]
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
}

func PullRemote() (TerraformState, error) {
	var tfstate TerraformState
	cmd := exec.Command("terraform", "state", "pull")
	var out bytes.Buffer
	cmd.Stdout = &out

	err := cmd.Run()
	if err != nil {
		return tfstate, err
	}
	json.Unmarshal(out.Bytes(), &tfstate)
	return tfstate, nil
}

func RemoveInstance(id string) (string, error) {
	cmd := exec.Command("terraform", "state", "rm", id)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	return out.String(), cmd.Run()
}

func ImportInstance(id, newResourceID string) (string, error) {
	cmd := exec.Command("terraform", "import", id, newResourceID)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	return out.String(), cmd.Run()
}

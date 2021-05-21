package state

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type ArrayVars []string

func (i *ArrayVars) String() string {
	return ""
}

func (i *ArrayVars) Set(value string) error {
	idx := strings.Index(value, "=")
	if idx == -1 {
		return fmt.Errorf("no '=' value in arg: %s", value)
	}

	*i = append(*i, "-var", value)
	return nil
}

type ArrayVarFiles []string

func (i *ArrayVarFiles) String() string {
	return fmt.Sprint(*i)
}

func (i *ArrayVarFiles) Set(list string) error {
	*i = append(*i, fmt.Sprintf("-var-file=%s", list))
	return nil
}

func RemoveInstance(id string) (string, error) {
	cmd := exec.Command("terraform", "state", "rm", id)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("terraform command \"terraform state rm %s\" failed : %v", id, err)
	}

	return out.String(), err
}

func ImportInstance(id, newResourceID string, vars ArrayVars, varfiles ArrayVarFiles) (string, error) {
	tfVars := append(vars, varfiles...)
	importVars := append(tfVars, id, newResourceID)
	cmdVars := append([]string{"import"}, importVars...)

	cmd := exec.Command("terraform", cmdVars...)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	err := cmd.Run()
	if err != nil {
		err = fmt.Errorf("terraform command \"terraform %s\" failed: %v", strings.Join(cmdVars, " "), err)
	}

	return out.String(), err
}

func (s *TerraformState) parseState(data []byte) error {
	return json.Unmarshal(data, s)
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
	err = tfstate.parseState(out.Bytes())

	return tfstate, err
}

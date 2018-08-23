package credentials

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	yaml "gopkg.in/yaml.v2"
)

// CredentialSet represents a collection of credentials
type CredentialSet struct {
	// Name is the name of the credentialset.
	Name string
	// Creadentials is a list of credential specs.
	Credentials []CredentialStrategy
}

// Load a CredentialSet from a file at a given path.
func Load(path string) (*CredentialSet, error) {
	cset := &CredentialSet{}
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return cset, err
	}
	return cset, yaml.Unmarshal(data, cset)
}

// Resolve looks up the credentials as described in Source, then copies the resulting value into Destination.
func (c *CredentialSet) Resolve() (map[string]Destination, error) {
	l := len(c.Credentials)
	res := make(map[string]Destination, l)
	for i := 0; i < l; i++ {
		src := c.Credentials[i].Source
		dest := c.Credentials[i].Destination
		fmt.Printf("%#v\n", src)
		// Precedence is Command, Path, EnvVar, Value
		switch {
		case src.Command != "":
			data, err := execCmd(src.Command)
			if err != nil {
				return res, err
			}
			dest.Value = string(data)
		case src.Path != "":
			data, err := ioutil.ReadFile(src.Path)
			if err != nil {
				return res, fmt.Errorf("credential %q: %s", c.Credentials[i].Name, err)
			}
			dest.Value = string(data)
		case src.EnvVar != "":
			println("LOOKING FOR ENV VAR", src.EnvVar)
			var ok bool
			dest.Value, ok = os.LookupEnv(src.EnvVar)
			if ok {
				break
			}
			fallthrough
		default:
			dest.Value = src.Value
		}
		res[c.Credentials[i].Name] = dest
	}
	return res, nil
}

func execCmd(cmd string) ([]byte, error) {
	parts := strings.Split(cmd, " ")
	c := parts[0]
	args := parts[1:]
	run := exec.Command(c, args...)

	return run.CombinedOutput()
}

// CredentialStrategy represents a source credential and the destination to which it should be sent.
type CredentialStrategy struct {
	Name        string      `json:"name" yaml:"name"`
	Source      Source      `json:"source,omitempty" yaml:"source,omitempty"`
	Destination Destination `json:"destination" yaml:"destination"`
}

// Source represents a strategy for loading a credential from local host.
type Source struct {
	//Type    string `json:"type"`
	Path    string `json:"path,omitempty" yaml:"path,omitempty"`
	Command string `json:"command,omitempty" yaml:"command,omitempty"`
	Value   string `json:"value,omitempty" yaml:"value,omitempty"`
	EnvVar  string `json:"env,omitempty" yaml:"env,omitempty"`
}

// Destination reprents a strategy for injecting a credential into an image.
type Destination struct {
	Path   string `json:"path,omitempty" yaml:"path,omitempty"`
	EnvVar string `json:"env,omitempty" yaml:"env,omitempty"`
	Value  string `json:"value,omitempty" yaml:"value,omitempty"`
}

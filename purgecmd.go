package main

import (
	"fmt"
	"os"
	"path"
)

// PurgeCmd stores the parsed command line arguments used to invoke the Purge command.
type PurgeCmd struct {
	Name string `arg help:"Name of the project to remove"`
}

// Run runs the Purge command which purges a project from the ghsec configuration.  This removes both the configuration file
// and the secret repository.
func (cmd *PurgeCmd) Run() error {
	config := LoadProjectConfigNoChecks(cmd.Name)
	configFile := ConfigFile(cmd.Name)
	repoDir := path.Join(ConfigDir(), config.RepoName)
	os.RemoveAll(repoDir)
	os.Remove(configFile)

	fmt.Printf("The project %v was successfully purged", cmd.Name)

	return nil
}

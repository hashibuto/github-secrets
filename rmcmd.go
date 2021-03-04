package main

import (
	"fmt"
	"os"
	"path"
)

// RmCmd stores the parsed command line arguments used to invoke the Rm command.
type RmCmd struct {
	Name     string `arg help:"Name of the project from which to remove a secret file"`
	Filename string `arg help:"Name of the secret file to remove"`
}

// Run runs the Rm command which removes a secret from the project
func (cmd *RmCmd) Run() error {
	config := LoadAndUpdateRepo(cmd.Name)
	repoDir := path.Join(ConfigDir(), config.RepoName)

	wd, err := os.Getwd()
	if err != nil {
		return err
	}
	os.Remove(path.Join(wd, cmd.Filename))

	for _, ext := range []string{"enc", "md5"} {
		filename := fmt.Sprintf("%v.%v", cmd.Filename, ext)
		err = ExecStreamOutput(repoDir, "git", "rm", filename)
		if err != nil {
			fmt.Printf("Unable to remove %v, please make sure it exists", filename)
			return err
		}
	}

	err = PushChanges(repoDir, config.Branch)
	if err != nil {
		return err
	}

	fmt.Println("Secret successfully removed")
	return nil
}

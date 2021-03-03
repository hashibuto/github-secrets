package main

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"os/exec"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

const githubUser = "git@ghub.com"

type InitCmd struct {
	Name      string `arg help:"Name of project to initialize"`
	GitHubURL string `arg help:"URL of the GitHub repository which stores the secrets"`
}

func (cmd *InitCmd) Run() error {
	ghUrl, err := url.Parse(cmd.GitHubURL)
	if err != nil {
		fmt.Println("Unable to parse URL: %v", cmd.GitHubURL)
		os.Exit(1)
	}

	if ghUrl.Host != "github.com" {
		fmt.Println("Not a github URL")
		os.Exit(1)
	}

	if len(ghUrl.Path) == 0 {
		fmt.Println("URL does not point to a repository")
		os.Exit(1)
	}
	pathRunes := []rune(ghUrl.Path)
	if pathRunes[0] == '/' {
		pathRunes = pathRunes[1:]
	}

	repoPath := string(pathRunes)
	repoPathParts := strings.Split(repoPath, "/")
	repoName := repoPathParts[len(repoPathParts)-1]

	fmt.Println("Initializing", cmd.Name)

	ghSecDir := ConfigDir()

	_, err = os.Stat(ghSecDir)
	if err != nil {
		os.Mkdir(ghSecDir, os.FileMode(0700))
	}

	// Delete any existing repository
	rmCmd := exec.Command("rm", "-rf", path.Join(ghSecDir, repoName))
	err = rmCmd.Run()
	if err != nil {
		panic(err)
	}

	// Prepare the ssh address for cloning the repository
	sshAddr := fmt.Sprintf("%v:%v.git", githubUser, repoPath)
	fmt.Println("Cloning secrets from", sshAddr)
	err = ExecStreamOutput(ghSecDir, "git", "clone", sshAddr)
	if err != nil {
		fmt.Println("Failed to clone remote secrets repository, aborting...")
		os.Exit(1)
	}

	secretConfig := &Config{
		RepoDir: repoName,
		EncKey:  "<replace with your 32 byte encryption key>",
	}

	fmt.Println("Writing configuration...")
	ymlData, err := yaml.Marshal(secretConfig)
	if err != nil {
		panic(err)
	}

	configLoc := ConfigFile(cmd.Name)
	err = ioutil.WriteFile(configLoc, ymlData, 0600)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Please edit the file %v and set your 32 byte encryption key\n", configLoc)

	return nil
}

package main

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path"
	"strings"

	"gopkg.in/yaml.v2"
)

const githubUser = "git@github.com"
const commitEmail = "ghsec@noreply.com"

// InitCmd stores the parsed command line arguments used to invoke the Init command.
type InitCmd struct {
	Name          string `arg help:"Name of project to initialize"`
	GitHubURL     string `arg help:"URL of the GitHub repository which stores the secrets"`
	CommitterName string `arg help:"Name to associate with commits from this machine"`
}

// Run runs the Init command which initializes the configuration environment and clones the pre-made
// remote git repository which will house the encrypted secrets
func (cmd *InitCmd) Run() error {
	configLoc := ConfigFile(cmd.Name)
	_, err := os.Stat(configLoc)
	if err == nil {
		// File already exists
		log.Fatal("A project by this name has already been initialized")
	}

	ghURL, err := url.Parse(cmd.GitHubURL)
	if err != nil {
		log.Fatal("Unable to parse URL:", cmd.GitHubURL)
	}

	if ghURL.Host != "github.com" {
		log.Fatal("Not a github URL")
	}

	if len(ghURL.Path) == 0 {
		log.Fatal("URL does not point to a repository")
	}
	pathRunes := []rune(ghURL.Path)
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
	err = os.RemoveAll(path.Join(ghSecDir, repoName))

	// Prepare the ssh address for cloning the repository
	sshAddr := fmt.Sprintf("%v:%v.git", githubUser, repoPath)
	fmt.Println("Cloning secrets from", sshAddr)
	err = ExecStreamOutput(ghSecDir, "git", "clone", sshAddr)
	if err != nil {
		log.Fatal("Failed to clone remote secrets repository, aborting...")
	}

	fmt.Println("Configuring committer info")
	repoDir := path.Join(ghSecDir, repoName)
	err = ExecStreamOutput(repoDir, "git", "config", "user.email", commitEmail)
	if err != nil {
		return err
	}
	err = ExecStreamOutput(repoDir, "git", "config", "user.name", cmd.CommitterName)
	if err != nil {
		return err
	}

	key := make([]byte, KeyBytes)
	_, err = rand.Read(key)
	if err != nil {
		return err
	}

	secretConfig := &Config{
		RepoName: repoName,
		EncKey:   base64.StdEncoding.EncodeToString(key),
		Branch:   DefaultBranch,
	}

	fmt.Println("Writing configuration...")
	ymlData, err := yaml.Marshal(secretConfig)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(configLoc, ymlData, 0600)
	if err != nil {
		return err
	}

	fmt.Println("Your configuration has been initialized at", configLoc)
	fmt.Println("If this is for an existing secrets repository, please edit the file and change the 'enckey' value and replace with you existing Base64 encoded token.")
	fmt.Println("If you are initializing a new repository, you may securely exchange the Base64 encoded key located at 'enckey' with your teammates.")

	return nil
}

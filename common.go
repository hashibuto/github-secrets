package main

import (
	"bufio"
	"crypto/md5"
	"encoding/base64"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path"

	"gopkg.in/yaml.v2"
)

const configDir = ".ghsec"
const configFilePattern = ".config.%v.yml"

// SecretPattern represents pattern used to locate secret files at the target location
const SecretPattern = "secret.*"

// DefaultBranch is the default branch on the repository
const DefaultBranch = "master"

// KeyBytes defines how many bytes are required for the AES256 encryption key
const KeyBytes = 32

// Config represents the configuration file structure after unmarshaling from YAML
type Config struct {
	RepoName string
	EncKey   string
	Branch   string
}

// RepoPath returns the full path to the repository identified in config
func (config *Config) RepoPath() string {
	return path.Join(ConfigDir(), config.RepoName)
}

// ConfigDir returns the configuration directory for ghsec
func ConfigDir() string {
	usr, err := user.Current()
	if err != nil {
		panic(err)
	}

	return path.Join(usr.HomeDir, configDir)
}

// ConfigFile returns the configuration file name for a given project
func ConfigFile(projectName string) string {
	return path.Join(ConfigDir(), fmt.Sprintf(configFilePattern, projectName))
}

// LoadAndUpdateRepo updates the underlying git repository with the latest remote changes.  Any pending changes (which should not exist)
// but could be left as artifacts from a previously failed operation, are reset prior to the update.  The configuration struct is returned
// if the operation is successful.
func LoadAndUpdateRepo(projectName string) *Config {
	fmt.Println("Loading project configuration")
	config := LoadProjectConfig(projectName)
	fmt.Println("Resetting any local changes")
	err := ExecStreamOutput(path.Join(ConfigDir(), config.RepoName), "git", "reset", "--hard", "HEAD")
	if err != nil {
		log.Fatal("Unable to reset repository changes")
	}
	fmt.Println("Pulling latest changes from repository")
	PullLatest(config)

	return config
}

// LoadProjectConfig returns a Config object representing the configuration of a given project
func LoadProjectConfig(projectName string) *Config {
	config := LoadProjectConfigNoChecks(projectName)

	// Pre-decode the encryption key in order to make sure it's what is expected and if not, abort
	encKey, err := base64.StdEncoding.DecodeString(config.EncKey)
	if err != nil {
		log.Fatal("Encryption key in project configuration file must be a standard Base64 encoded string")
	}

	if len(encKey) != KeyBytes {
		log.Fatalf("Your encryption key must be %v bytes long in order to support AES256 encryption, please fix it\n", KeyBytes)
	}

	return config
}

// LoadProjectConfigNoChecks returns a Config object representing the configuration of a given project without performing
// any validation of the contents of the project (aside from requiring its existance and well formed configuration file)
func LoadProjectConfigNoChecks(projectName string) *Config {
	configFile := ConfigFile(projectName)
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("Unable to load configuration file", configFile)
		log.Fatal("Please ensure that the project has been initialized")
	}

	config := Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("Error parsing config yaml:", err)
	}

	return &config
}

// ExecStreamOutput executes a command at the given location and streams the output from both
// stderr and stdout to the stdout
func ExecStreamOutput(location string, command ...string) error {
	mainCmd := command[0]
	cmdArgs := command[1:]

	cmd := exec.Command(mainCmd, cmdArgs...)
	cmd.Dir = location
	cmd.Stderr = cmd.Stdout

	output, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(output)
	cmd.Start()
	for scanner.Scan() {
		line := scanner.Text()
		fmt.Println(line)
	}

	return cmd.Wait()
}

// PullLatest pulls the latest changes from the repository indicated in the configuration
func PullLatest(config *Config) {
	RepoName := path.Join(ConfigDir(), config.RepoName)

	err := ExecStreamOutput(RepoName, "git", "pull")
	if err != nil {
		log.Fatal("Failed to pull from git repository", err)
	}
}

// PushChanges commits and pushes changes to the remote git repository
func PushChanges(repoDir string, branch string) error {
	err := ExecStreamOutput(repoDir, "git", "commit", "-am", "\"Updating secrets\"")
	if err != nil {
		return err
	}

	err = ExecStreamOutput(repoDir, "git", "push", "origin", branch)
	if err != nil {
		return err
	}

	return nil
}

// MD5FileHash returns a standard Base64 encoded string representation of the MD5 hash of the contents of the target file
func MD5FileHash(filename string) (string, error) {
	hash := md5.New()
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	io.Copy(hash, file)
	return base64.StdEncoding.EncodeToString(hash.Sum(nil)), nil
}

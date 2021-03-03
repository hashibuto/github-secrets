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

type Config struct {
	RepoDir string
	EncKey  string
}

// RepoPath returns the full path to the repository identified in config
func (config *Config) RepoPath() string {
	return path.Join(ConfigDir(), config.RepoDir)
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

// LoadProjectConfig returns a Config object representing the configuration of a given project
func LoadProjectConfig(projectName string) *Config {
	configFile := ConfigFile(projectName)
	data, err := ioutil.ReadFile(configFile)
	if err != nil {
		fmt.Println("Unable to load configuration file", configFile)
		fmt.Println("Please ensure that the project has been initialized")
		os.Exit(1)
	}

	config := Config{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatal("Error parsing config yaml:", err)
	}

	if len(config.EncKey) != 32 {
		fmt.Println("Your encryption key is not 32 bytes long, please fix it")
		os.Exit(1)
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
	repoDir := path.Join(ConfigDir(), config.RepoDir)

	err := ExecStreamOutput(repoDir, "git", "pull")
	if err != nil {
		log.Fatal("Failed to pull from git repository", err)
	}
}

// MD5FileHash returns a
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

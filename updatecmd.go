package main

import (
	"crypto/aes"
	"crypto/cipher"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type UpdateCmd struct {
	Name string `arg help:"Name of the project on which to update secrets"`
}

func (cmd *UpdateCmd) Run() error {
	fmt.Println("Loading project configuration")
	config := LoadProjectConfig(cmd.Name)
	fmt.Println("Pulling latest changes from repository")
	PullLatest(config)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Println("Searching in", cwd)

	matches, err := filepath.Glob("secret.*")
	if err != nil {
		return err
	}

	for _, filename := range matches {
		nameParts := strings.Split(filename, ".")
		ext := nameParts[len(nameParts)-1]
		if ext != "gpg" && ext != "md5" {
			fmt.Println("Found", filename)
			filePath := path.Join(cwd, filename)

			hash, err := MD5FileHash(filePath)
			if err != nil {
				return err
			}
			fmt.Println("Checksum", hash)
			md5Filename := fmt.Sprintf("%v.md5", filename)
			md5Path := path.Join(config.RepoPath(), md5Filename)
			oldMD5 := ""
			_, err = os.Stat(md5Path)
			if err == nil {
				fmt.Println("Checking existing MD5 hash")
				oldMD5Bytes, err := ioutil.ReadFile(md5Path)
				if err != nil {
					return err
				}
				oldMD5 = string(oldMD5Bytes)
			} else {
				fmt.Println("File appears to be new")
			}

			if oldMD5 != hash {
				unencData, err := ioutil.ReadFile(filePath)
				if err != nil {
					return err
				}

				// Hash has changed, write the encrypted file and update the hash
				block, err := aes.NewCipher([]byte(config.EncKey))
				if err != nil {
					return err
				}

				gcm, err := cipher.NewGCM(block)
				if err != nil {
					return err
				}

				nonce := make([]byte, gcm.NonceSize())
				_, err = rand.Read(nonce)
				if err != nil {
					return err
				}

				cipherText := gcm.Seal(nonce, nonce, unencData, nil)
				encFilename := fmt.Sprintf("%v.enc", filename)
				encFilePath := path.Join(config.RepoPath(), encFilename)
				fmt.Println("Writing encrypted file", encFilePath)

				err = ioutil.WriteFile(encFilePath, cipherText, os.FileMode(0664))
				if err != nil {
					return err
				}

				fmt.Println("Writing MD5 file", md5Path)
				err = ioutil.WriteFile(md5Path, []byte(hash), os.FileMode(0664))
				if err != nil {
					return err
				}
			} else {
				fmt.Println("Checksum matches existing, skipping...")
			}
		}
	}

	repoDir := path.Join(ConfigDir(), config.RepoDir)

	filters := ["*.enc", "*.md5"]
	for filter := range filters {
		err := ExecStreamOutput(repoDir, "git", "add", filter)
		if err != nil {
			return err
		}
	}

	err := ExecStreamOutput(repoDir, "git", "commit", "-am", "\"Updating secrets\"")
	if err != nil {
		return err
	}

	err := ExecStreamOutput(repoDir, "git", "push", "origin", "master")
	if err != nil {
		return err
	}

	return nil
}

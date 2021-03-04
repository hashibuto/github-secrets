package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// UpdateCmd stores the parsed command line arguments used to invoke the Update command.
type UpdateCmd struct {
	Name string `arg help:"Name of the project on which to update secrets"`
}

// Run runs the Update command which attempts to update the encrypted secrets in the git repository.  Only
// modified secrets will be encrypted and committed to the repository.
func (cmd *UpdateCmd) Run() error {
	config := LoadAndUpdateRepo(cmd.Name)

	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	fmt.Println("Searching in", cwd)

	matches, err := filepath.Glob(SecretPattern)
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

				// This has already been vetted, so no need to check for error
				key, _ := base64.StdEncoding.DecodeString(config.EncKey)

				// Hash has changed, write the encrypted file and update the hash
				block, err := aes.NewCipher(key)
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

	repoDir := path.Join(ConfigDir(), config.RepoName)

	filters := []string{"*.enc", "*.md5"}
	for _, filter := range filters {
		err := ExecStreamOutput(repoDir, "git", "add", filter)
		if err != nil {
			return err
		}
	}

	return PushChanges(repoDir, config.Branch)
}

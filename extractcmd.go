package main

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
)

// ExtractCmd stores the parsed command line arguments used to invoke the Extract command
type ExtractCmd struct {
	Name string `arg help:"Name of the project for which to extract secrets"`
}

// Run runs the Extract command which extracts secrets from the secret store to the CWD
func (cmd *ExtractCmd) Run() error {
	config := LoadAndUpdateRepo(cmd.Name)

	RepoName := path.Join(ConfigDir(), config.RepoName)
	filePattern := path.Join(RepoName, "secret.*.enc")

	matches, err := filepath.Glob(filePattern)
	if err != nil {
		return err
	}

	for _, filePath := range matches {
		fmt.Println("Decrypting", filePath)
		encData, err := ioutil.ReadFile(filePath)
		if err != nil {
			return err
		}

		// This has already been vetted, so no need to check for error
		key, _ := base64.StdEncoding.DecodeString(config.EncKey)

		block, err := aes.NewCipher(key)
		if err != nil {
			return err
		}

		gcm, err := cipher.NewGCM(block)
		if err != nil {
			return err
		}

		nonce := encData[:gcm.NonceSize()]
		encData = encData[gcm.NonceSize():]
		unencData, err := gcm.Open(nil, nonce, encData, nil)
		if err != nil {
			return err
		}

		_, fileName := path.Split(filePath)
		// Remove the .enc extension
		origName := fileName[:len(fileName)-4]
		fmt.Println("Writing", origName)

		// Write secret files back as accessible only by the current user.  If this doesn't fit a
		// given workflow, the user can opt to alter the file permissions after execution of this
		// command.  Additionally, it is assumed, not enforced, that the project will include a
		// .gitignore which prevents the accidental commit of secrets files
		err = ioutil.WriteFile(origName, unencData, os.FileMode(0600))
		if err != nil {
			return err
		}

		fmt.Println("Extraction complete")
	}

	return nil
}

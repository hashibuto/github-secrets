# github-secrets
Utility for management and storage of encrypted secrets within GitHub

## Synopsis
Projects often rely on sensitive data such as API access tokens, private keys, etc. which must be securely stored, yet remain available to the project or deployment for use.  This project attempts to solve that problem by facilitating encrypted secret storage using GitHub as the storage location.  Secrets are encrypted and decrypted using AES256 based on a symmetric key, which can be shared privately among project team members.

This project does not have a concept of ACLs - this is a single access gets all secrets system, though further access control can be applied if secrets are separately encrypted prior to being re-encrypted and stored in this system.

## Prerequisites
This has been tested on Linux, but may or may not run on other operating systems.  Git must be preinstalled.

## Creating a new secret store
First thing in getting set up, is to create a new github repository which will be used to store the encrypted secrets.

Next, initialize your local secrets configuration for your project using the `ghsec` binary, as follows:

```
ghsec init <project_name> <github_repository> <my_committer_name>
```

Below is fleshed out example of the above command:

```
ghsec init myproject https://www.github.com/myname/myproject "John Donut"
```

This will initialize a private directory in the current user's home directory called `.ghsec`.  Within will exist the configuration file, and local git repository which houses the secrets.  You may edit the file `.config.<project_name>.yml` at that location, in order to change certain details such as the default branch in the repository, or the encryption key.  The initialized encryption key is perfectly fine to use, if this is the first instance being created, of the secret store for this project.  If the project was initialized elsewhere, <b>it is necessary to update the `enckey` field with the previously generated token</b>.

## Adding/Updating secrets
In order to add or modify secret files, simply run `ghsec` from your own project directory where your secrets live.  `ghsec` will look for the pattern `secret.*` within that directory and automatically add them to the repository by executing the following command:

```
ghsec update <project_name>
```

<b>Caution:</b> The update command will indescriminantly overwrite any pending updates from remote.  For this reason, it is ideal that a single person perform updates to the secrets.  For this reason it could be considered a wise decision to allow write access only to a few (or a single) select user(s) to the secrets repository.

## Extracting secrets
As a consumer of the secrets store, you will want a filter set up in `.gitignore` which targets `secret.*` (if the extraction location is within a git repository).  Run the following command from the target directory where secrets should be extracted:

```
ghsec extract <project_name>
```

## Removing a secret file
Should a secret file become unnecessary, simply remove it from the repository using the following command:
```
ghsec rm <project_name> <secret_filename>
```

The secret filename refers to the unencrypted filename, not the encrypted name stored in git.

## Removing a project configuration
If a secret project configuration is no longer needed, it can be removed by the following command:
```
ghsec purge <project_name>
```

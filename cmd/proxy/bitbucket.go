package main

import (
	"errors"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"golang.org/x/crypto/ssh"

	"gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	gitssh "gopkg.in/src-d/go-git.v4/plumbing/transport/ssh"
)

const DirStructure = `(?P<fullpath>.*\/)(?P<environment>[\w|\W].*)\/(?P<filename>.*.tf)`

type environment struct {
	Environment string
}

func gitClone(c string) (string, error) {
	repoURL := "git@bitbucket.org:ovoeng/terraform.git"
	cloneDir := "/opt/terraform"

	var r *git.Repository
	var err error

	sshPath := os.Getenv("HOME") + "/.ssh/id_rsa"
	sshKey, _ := ioutil.ReadFile(sshPath)
	signer, _ := ssh.ParsePrivateKey(sshKey)
	auth := &gitssh.PublicKeys{
		User:   "git",
		Signer: signer,
	}

	// check if the folder already exists
	if _, err := os.Stat(cloneDir); err != nil {
		if os.IsNotExist(err) {

			// Clone the given repository to the given directory
			log.Printf("git clone %s %s --recursive", repoURL, cloneDir)

			r, err = git.PlainClone(cloneDir, false, &git.CloneOptions{
				URL:               repoURL,
				RecurseSubmodules: git.DefaultSubmoduleRecursionDepth,
				Auth:              auth,
			})

			if err != nil {
				return "", err
			}
		}
	} else {
		log.Printf("Found existing directory %s", cloneDir)

		r, err = git.PlainOpen(cloneDir)

		if err != nil {
			return "", err
		}
	}

	// fetch the repository
	_ = r.Fetch(&git.FetchOptions{
		Auth:  auth,
		Force: true,
	})

	// get the commit hash
	log.Printf("git rev-parse : %s", c)
	revParseCmd := exec.Command("git", "rev-parse", c)
	revParseCmd.Dir = cloneDir
	outputRevParse, err := revParseCmd.CombinedOutput()

	hashRev := strings.Trim(string(outputRevParse), "\n")

	// checking out
	w, err := r.Worktree()
	if err != nil {
		return "", nil
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hashRev),
	})

	if err != nil {
		return "", nil
	}

	log.Printf("git pull origin")

	_ = w.Pull(&git.PullOptions{
		RemoteName: "origin",
		Auth:       auth,
	})

	log.Printf("checking out to commit hash %s", hashRev)
	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(hashRev),
	})

	env := getEnvironment()

	if len(env) > 1 {
		err = errors.New("Found multiple environment in one commit")
		log.Printf("Found multiple environment in one commit")
		log.Printf("Environment found :")
		for _, i := range env {
			log.Printf("%s", i)
		}
		return "", err
	}

	log.Printf("Found environment : %s", env)

	environment := strings.Join(env, ",")

	return environment, nil
}

func getEnvironment() []string {
	cloneDir := "/opt/terraform"
	var matcher *regexp.Regexp
	//var finalDir string
	matcher = regexp.MustCompile(DirStructure)

	repo, _ := git.PlainOpen(cloneDir)
	ref, _ := repo.Head()
	commit, _ := repo.CommitObject(ref.Hash())
	fileStats := object.FileStats{}

	fileStats, _ = commit.Stats()

	filePaths := []string{}

	for _, fileStat := range fileStats {
		filePaths = append(filePaths, fileStat.Name)
	}

	f := make(map[string]string)
	pr := []string{}
	for _, filePath := range filePaths {
		if !strings.Contains(filePath, "/") {
			continue
		}

		// check if the changes is in the same directory
		matches := matcher.FindStringSubmatch(filePath)

		if len(matches) == 0 {
			continue
		} else {
			for i, name := range matcher.SubexpNames() {
				if name == "" {
					continue
				}
				f[name] = matches[i]
			}
		}

		for key, value := range f {
			if key == "environment" {
				pr = append(pr, value)
			}
		}
	}
	env := removeDupes(pr)

	return env
}

func removeDupes(folder []string) []string {
	e := map[string]bool{}

	for i := range folder {
		e[folder[i]] = true
	}

	result := []string{}
	for key, _ := range e {
		result = append(result, key)
	}

	return result
}

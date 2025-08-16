//go:build mage

package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/bitfield/script"
	"github.com/magefile/mage/sh"
)

// Update updates the dependencies of all modules.
// It only updates direct dependencies within the same major version, like `go get` with `@latest` does.
// It doesn't update transitive dependencies, like `go get -u` does.
// It also runs `go mod tidy` for all modules after updating.
func Update() error {
	switch runtime.GOOS {
	case "windows":
		return sh.Run("pwsh.exe", "./build/update-deps.ps1")
	case "darwin":
		fallthrough
	case "linux":
		return sh.Run("./build/update-deps.sh")
	}
	return errors.New("your OS is not supported")
}

// Build builds all modules.
func Build() error {
	switch runtime.GOOS {
	case "windows":
		return sh.Run("pwsh.exe", "./build/build.ps1")
	case "darwin":
		fallthrough
	case "linux":
		return sh.Run("./build/build.sh")
	}
	return errors.New("your OS is not supported")
}

// Test tests the given module. Pass "all" to test all modules.
func Test(module string) error {
	if module == "all" {
		// Helper packages and examples currently don't have tests, so currently for *all* tests we can just iterate all `gokv.Store` implementations
		// TODO: Add tests for helper and example packages, then change this behavior.
		impls, err := script.File("./build/implementations").Slice()
		if err != nil {
			return err
		}
		for _, impl := range impls {
			err = testImpl(impl)
			if err != nil {
				return err
			}
		}
		return nil
	}

	switch module {
	case "encoding", "sql", "test", "util":
		return errors.New("module " + module + " doesn't have any tests")
	case "examples":
		return errors.New("examples don't have any tests")
	}

	i, err := script.File("./build/implementations").Match(module).CountLines()
	if err != nil {
		return err
	}
	if i == 0 {
		return errors.New("module from parameter not found")
	}

	return testImpl(module)
}

// Clean cleans the build/test output, like coverage.txt files
func Clean() error {
	p := script.FindFiles(".").
		Match("coverage.txt").
		ExecForEach("rm ./{{.}}") // On Windows `rm` works as it's an alias for Remove-Item
	p.Wait()
	return p.Error()
}

// Replace replaces the tagged/versioned gokv dependencies with the local version.
// This is useful before tagging/releasing a new version of a helper package,
// to ensure all store implementations work with those latest helper package versions.
func Replace() error {
	rootDir, err := os.Getwd()
	if err != nil {
		return err
	}

	impls, err := script.File("./build/implementations").Slice()
	if err != nil {
		return err
	}
	replaceText := `
replace (
	github.com/philippgille/gokv => ../
	github.com/philippgille/gokv/encoding => ../encoding
	github.com/philippgille/gokv/test => ../test
	github.com/philippgille/gokv/util => ../util
	github.com/philippgille/gokv/sql => ../sql
)
`
	for _, impl := range impls {
		modFile := filepath.Join(".", impl, "go.mod")
		content, err := os.ReadFile(modFile)
		if err != nil {
			return err
		}
		newContent := string(content) + replaceText

		err = os.WriteFile(modFile, []byte(newContent), os.ModePerm)
		if err != nil {
			return err
		}

		err = os.Chdir(filepath.Join(".", impl))
		if err != nil {
			return err
		}
		err = sh.Run("go", "mod", "tidy")
		if err != nil {
			return err
		}
		err = os.Chdir(rootDir)
		if err != nil {
			return err
		}
	}
	return nil
}

// Lint lints the given module. Pass "all" to lint all modules.
func Lint(module string) error {
	rootDir, err := os.Getwd()
	if err != nil {
		return err
	}

	var dirs []string
	if module == "all" {
		// Gather every module directory
		mods, err := script.FindFiles(".").Match("go.mod").Slice()
		if err != nil {
			return err
		}
		for _, mod := range mods {
			dirs = append(dirs, filepath.Dir(mod))
		}
	} else {
		// Validate specified module
		modDir := filepath.Join(".", module)
		modFile := filepath.Join(modDir, "go.mod")
		if _, err := os.Stat(modFile); err != nil {
			if os.IsNotExist(err) {
				return errors.New("module " + module + " not found")
			}
			return err
		}
		dirs = []string{modDir}
	}

	// Run golangci-lint in each determined directory
	for _, dir := range dirs {
		fmt.Printf("Running golangci-lint in %s\n", dir)

		if err := os.Chdir(dir); err != nil {
			return err
		}

		cmd := "golangci-lint run"
		// golangci-lint doesn't look for the config file in the repo root, so we
		// have to pass it explicitly.
		cmd += " --config " + filepath.Join(rootDir, ".golangci.yml")
		if filepath.Base(dir) == "magefiles" {
			cmd += " --build-tags=mage"
		}

		out, err := script.Exec(cmd).String()
		fmt.Println(out)
		if err != nil {
			return err
		}

		if err := os.Chdir(rootDir); err != nil {
			return err
		}
	}

	return nil
}

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func newCheckCmd() *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   "check",
		Short: "Check dependencies are installed",
		Long:  "Check that dependencies are installed and within required versions",
		PreRun: func(_ *cobra.Command, _ []string) {
			initLogging()
		},
		Run: func(cmd *cobra.Command, args []string) {
			var (
				checkGit = func() error {
					// Example version output:
					//     git version 2.17.1

					log.Debug("Checking git version")

					cmd := exec.Command("git", "--version")
					out, err := cmd.CombinedOutput()
					if err != nil {
						return fmt.Errorf("git: version check command failed: %s", err)
					}
					out = bytes.Trim(out, " \t\r\n")
					log.Debugf("Detected git version: %v", string(out))

					pieces := strings.Split(string(out), " ")
					if len(pieces) < 3 {
						return fmt.Errorf("git: version check command output not recognized, output=%v", string(out))
					}
					versions := strings.Split(pieces[2], ".")
					if len(versions) < 3 {
						return fmt.Errorf("git: version string not recognized, value=%v", pieces[2])
					}
					version := make([]int64, len(versions))
					for i, vs := range versions {
						v, _ := strconv.ParseInt(vs, 0, 0)
						version[i] = v
					}
					if version[0] < 2 {
						return fmt.Errorf("git: major version must be >= 2 but actual installed major version=%v", version[0])
					}
					if version[0] == 2 && version[1] < 3 {
						return fmt.Errorf("git: for 2.x, minor version must be >= 3 but actual installed minor version =%v", version[1])
					}
					return nil
				}

				checkOpenSSL = func() error {
					log.Debug("Checking OpenSSL version")

					cmd := exec.Command("openssl", "version")
					out, err := cmd.CombinedOutput()
					if err != nil {
						return fmt.Errorf("openssl: version check command failed: %s", err)
					}
					out = bytes.Trim(out, " \t\r\n")
					log.Debugf("Detected openssl version: %v", string(out))
					return nil
				}

				checkFns = []func() error{
					checkGit,
					checkOpenSSL,
				}
				numIssues = 0
			)

			for _, checkFn := range checkFns {
				if err := checkFn(); err != nil {
					log.Error(err)
					numIssues++
				}
			}
			log.Infof("Checks passed: %v / %v", len(checkFns)-numIssues, len(checkFns))
			if numIssues > 0 {
				log.Fatalf("Not all checks passed")
			}
		},
	}
	return checkCmd
}

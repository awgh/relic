/*
 * Copyright (c) SAS Institute Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package remotecmd

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"strings"

	"gerrit-pdt.unx.sas.com/tools/relic.git/lib/binpatch"
	"github.com/spf13/cobra"
)

var SignCmd = &cobra.Command{
	Use:   "sign",
	Short: "Sign a package using a remote signing server",
	RunE:  signCmd,
}

var (
	argKeyAlias string
)

func init() {
	RemoteCmd.AddCommand(SignCmd)
	SignCmd.Flags().StringVarP(&argKeyName, "key", "k", "", "Name of key on remote server to use")
	SignCmd.Flags().StringVarP(&argFile, "file", "f", "", "Input file to sign")
	SignCmd.Flags().StringVarP(&argOutput, "output", "o", "", "Output file. Defaults to same as --file.")
	SignCmd.Flags().StringVar(&argKeyAlias, "key-alias", "RELIC", "Alias to use for signed manifests (JAR only)")
}

func signCmd(cmd *cobra.Command, args []string) (err error) {
	if argFile == "" || argKeyName == "" {
		return errors.New("--file and --key are required")
	}
	if argOutput == "" {
		argOutput = argFile
	}
	exten := strings.ToLower(path.Ext(argFile))
	if exten == ".jar" {
		return signJar()
	}
	// open for writing so in-place patch works
	infile, err := os.OpenFile(argFile, os.O_RDWR, 0)
	if err != nil {
		return err
	}

	values := url.Values{}
	values.Add("key", argKeyName)
	values.Add("filename", path.Base(argFile))

	response, err := callRemote("sign", "POST", &values, infile)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.Header.Get("Content-Type") == "application/x-binary-patch" {
		blob, err := ioutil.ReadAll(response.Body)
		if err != nil {
			return err
		}
		patch, err := binpatch.Load(blob)
		if err != nil {
			return err
		}
		err = patch.Apply(infile, argOutput)
	} else {
		infile.Close()
		err = writeOutput(argOutput, response.Body)
	}
	if err == nil {
		fmt.Fprintf(os.Stderr, "Signed %s\n", argFile)
	}
	return
}

func writeOutput(path string, src io.Reader) error {
	if argOutput == "-" {
		_, err := io.Copy(os.Stdout, src)
		return err
	} else {
		outfile, err := os.Create(path)
		if err != nil {
			return err
		}
		_, err = io.Copy(outfile, src)
		outfile.Close()
		return err
	}
}
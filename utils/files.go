/*Copyright [2023] [Alejandro Escanero Blanco <aescanero@disasterproject.com>]

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.*/

package utils

import (
	"io"
	"os"
	"path/filepath"
)

func CopyFiles(origs []string, dest string) error {
	for _, file := range origs {
		err := CopyFile(file, dest+"/"+filepath.Base(file))
		if err != nil {
			return err
		}
	}
	return nil
}

func CopyFile(orig, copy string) error {

	source, err := os.Open(orig)
	if err != nil {
		return err
	}

	defer source.Close()

	destination, err := os.Create(copy)
	if err != nil {
		return err
	}

	defer destination.Close()

	BUFFERSIZE := 4096
	buf := make([]byte, BUFFERSIZE)
	for {
		n, err := source.Read(buf)
		if err != nil && err != io.EOF {
			return err
		}
		if n == 0 {
			break
		}

		if _, err := destination.Write(buf[:n]); err != nil {
			return err
		}
	}
	return nil
}

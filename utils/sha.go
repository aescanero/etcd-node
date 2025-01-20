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
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"log"
)

type Encode struct{}

func (enc Encode) MakeSSHAEncode(pass []byte) (passEncode []byte) {
	salt := make([]byte, 4)
	rand.Read(salt)
	sha := sha1.New()
	sha.Write(pass)
	sha.Write(salt)
	h := sha.Sum(nil)
	return append(h, salt...)
}

func (enc Encode) MakeSSHA256Encode(pass []byte) (passEncode []byte) {
	salt := make([]byte, 4)
	rand.Read(salt)
	sha := sha256.New()
	sha.Write(pass)
	sha.Write(salt)
	h := sha.Sum(nil)
	return append(h, salt...)
}

func (enc Encode) Matches(passwordSSHAB64, rawPassPhrase []byte, debug bool) bool {
	//strip the {SSHA}
	if debug {
		log.Printf("pass: %s\n", string(rawPassPhrase))
		log.Printf("passSSHAB64: %s\n", string(passwordSSHAB64))
	}
	eppS := string(passwordSSHAB64)[6:]
	if debug {
		log.Printf("eppS: %s\n", eppS)
	}
	hash, err := base64.StdEncoding.DecodeString(eppS)
	if err != nil {
		log.Fatal(err)
	}
	if debug {
		log.Printf("hash: %s\n", string(hash))
	}
	salt := hash[len(hash)-4:]
	if debug {
		log.Printf("salt: %s\n", string(salt))
	}
	sha := sha1.New()
	sha.Write(rawPassPhrase)
	sha.Write(salt)
	sum := sha.Sum(nil)
	if debug {
		log.Printf("sum: %s\n", string(sum))
	}
	if debug {
		log.Printf("hash: %s\n", string(hash[:len(hash)-4]))
	}
	return bytes.Equal(sum, hash[:len(hash)-4])
}

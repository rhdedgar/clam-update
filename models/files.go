/*
Copyright 2019 Doug Edgar.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package models

import "time"

// VerifiedFiles contains a slice of files along with their SHA256 hash checksums
type VerifiedFiles struct {
	LocalFiles map[string]LocalFile `json:"local_files"`
}

type LocalFile struct {
	Name     string    `json:"name"`
	Checksum string    `json:"checksum"`
	ModTime  time.Time `json:"mod_time"`
}

// VerifiedFiles returns a new empty VerifiedFiles struct.
func NewVerifiedFiles(mLen int) *VerifiedFiles {
	return &VerifiedFiles{
		LocalFiles: make(map[string]LocalFile, mLen),
	}
}

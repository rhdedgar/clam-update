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

// AppSecrets holds needed config info for the application to function
type AppSecrets struct {
	BucketName    string   `json:"signature_mirror_bucket"`
	BucketRegion  string   `json:"signature_bucket_region"`
	BucketKey     string   `json:"signature_bucket_key"`
	BucketKeyID   string   `json:"signature_bucket_key_id"`
	ContentFiles  []string `json:"signature_config_files"`
	TimestampPath string   `json:"ocav_timestamp_path"`
}

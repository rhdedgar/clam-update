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

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/rhdedgar/clam-update/models"
)

// upload gets timestamps of all files in the bucket, compares with local
// file timestamps, then uploads if the local files are newer
func upload(bucket string, fileList []string) {
	timeMap := make(map[string]time.Time)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)

	uploader := s3manager.NewUploader(sess)
	svc := s3.New(sess)

	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		fmt.Printf("Unable to list items in bucket %v, %v\n", bucket, err)
	}

	if err != nil {
		fmt.Println("Unable to open new AWS session: ", err)
	}

	// Add bucket file timestamps to a Map if the files are also in our file list
	for _, bucketItem := range resp.Contents {
		for _, listItem := range fileList {
			if *bucketItem.Key == listItem {
				timeMap[*bucketItem.Key] = *bucketItem.LastModified
			}
		}
	}

	for _, fileName := range fileList {
		filePath := path.Join("/var/lib/clamav/", fileName)

		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Println("Error getting last modification time from: ", filePath)
		}

		// Check if our local file was modified more recently than the bucket's copy
		if info.ModTime().After(timeMap[fileName]) {
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("Unable to open file %v: %v\n", fileName, err)
				continue
			}
			defer file.Close()

			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),

				Key: aws.String(filePath),

				Body: file,
			})
			if err != nil {
				fmt.Printf("Unable to upload %v to %v, %v\n", filePath, bucket, err)
				break
			}

			fmt.Printf("Successfully uploaded %v to %v\n", filePath, bucket)
		}
	}
}

func main() {
	var appSecrets models.AppSecrets

	filePath := os.Getenv("SECRET_CONFIG")
	fileBytes, err := ioutil.ReadFile(filePath)

	if err != nil {
		fmt.Println("Error loading secrets json: ", err)
	}

	err = json.Unmarshal(fileBytes, &appSecrets)
	if err != nil {
		fmt.Println("Error Unmarshaling secrets json: ", err)
	}

	os.Setenv("AWS_SHARED_CREDENTIALS_FILE", appSecrets.OcavCredsFile)

	upload(appSecrets.OcavS3Bucket, appSecrets.OcavOpsFiles)
}

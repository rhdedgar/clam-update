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
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/rhdedgar/clam-update/models"
)

var (
	appSecrets models.AppSecrets
	//verifiedFiles = map[string]string{}
	checksumCache = &models.VerifiedFiles{}
)

func getSession() (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(appSecrets.BucketRegion),
		Credentials: credentials.NewStaticCredentials(
			appSecrets.BucketKeyID,
			appSecrets.BucketKey,
			""),
	})
	if err != nil {
		return sess, fmt.Errorf("Unable to create new AWS session: %v\n", err)
	}

	return sess, nil
}

// upload gets timestamps of all files in the bucket, compares with local
// file timestamps, then uploads if the local files are newer
/*
func upload(bucket, fileDir string, fileList []string) error {
	timeMap := make(map[string]time.Time)

	sess, err := getSession()
	if err != nil {
		return fmt.Errorf("Error returned from getSession: %v\n", err)
	}

	uploader := s3manager.NewUploader(sess)
	svc := s3.New(sess)

	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(bucket)})
	if err != nil {
		return fmt.Errorf("Unable to list items in bucket %v, %v\n", bucket, err)
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
		filePath := path.Join(fileDir, fileName)

		info, err := os.Stat(filePath)
		if err != nil {
			fmt.Printf("Error getting last modification time from: %v\n", filePath)
			continue
		}

		// Check if our local file was modified more recently than the bucket's copy
		if info.ModTime().After(timeMap[fileName]) {
			file, err := os.Open(filePath)
			if err != nil {
				fmt.Printf("Unable to open file %v: %v\n", filePath, err)
				continue
			}
			defer file.Close()

			// BEGIN HASH
			sha := sha256.New()

			_, err = io.Copy(sha, file)
			if err != nil {
				return fmt.Errorf("Error io.Copying file %v: %v\n", fileName, err)
			}

			// TODO: if this string != the sha sum for the map item we read from the server checksum map returner
			checksum := hex.EncodeToString(sha.Sum(nil))

			if checksum != "" {
				verifiedFiles[fileName] = checksum
			}
			// END HASH

			_, err = uploader.Upload(&s3manager.UploadInput{
				Bucket: aws.String(bucket),

				Key: aws.String(fileName),

				Body: file,
			})
			if err != nil {
				return fmt.Errorf("Unable to upload %v to %v, %v\n", filePath, bucket, err)
			}

			fmt.Printf("Successfully uploaded %v to %v\n", filePath, bucket)
		}
	}
	return nil
}
*/

func upload(bucket, fileDir string, fileList []string) error {
	for _, fileName := range fileList {
		filePath := path.Join(fileDir, fileName)

		file, err := os.Open(filePath)
		if err != nil {
			fmt.Printf("Unable to open file %v: %v\n", filePath, err)
			continue
		}
		defer file.Close()

		// BEGIN HASH
		sha := sha256.New()

		//fileBytes, err := ioutil.ReadFile(fileName)
		//if err != nil {
		//	fmt.Println("reading error", err)
		//}

		//sha.Write(data)
		data := io.TeeReader(file, sha)
		checksum := hex.EncodeToString(sha.Sum(nil))

		//_, err = io.Copy(sha, data)
		//if err != nil {
		//		return fmt.Errorf("Error io.Copying file %v: %v\n", fileName, err)
		//	}

		// TODO: if this string != the sha sum for the map item we read from the server checksum map returner
		//checksum := hex.EncodeToString(sha.Sum(nil))

		if checksum == "" {
			continue
		}
		// END HASH

		/*_, err = uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(bucket),

			Key: aws.String(fileName),

			Body: data,
		})
		*/
		err = uploadSingle(bucket, fileName, data)
		if err != nil {
			return fmt.Errorf("Unable to upload %v to %v, %v\n", filePath, bucket, err)
		}

		fmt.Printf("Successfully uploaded %v to %v\n", filePath, bucket)
	}
	return nil
}

func uploadSingle(bucket, fileName string, payload interface{}) error {
	sess, err := getSession()
	if err != nil {
		return fmt.Errorf("Error returned from getSession: %v\n", err)
	}

	uploader := s3manager.NewUploader(sess)

	b := new(bytes.Buffer)

	err = json.NewEncoder(b).Encode(payload)
	if err != nil {
		return fmt.Errorf("Error encoding to bytes.Buffer: %v\n", err)
	}

	_, err = uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(fileName),
		Body:   b,
	})
	if err != nil {
		return fmt.Errorf("Unable to upload %v to %v, %v\n", fileName, bucket, err)
	}

	return nil
}

func loadConfigFile(filePath string, dest interface{}) error {
	fileBytes, err := ioutil.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("Error loading secrets json from:  %v %v\n", filePath, err)
	}

	err = json.Unmarshal(fileBytes, dest)
	if err != nil {
		return fmt.Errorf("Error Unmarshaling secrets json: %v\n", err)
	}
	return nil
}

// listBucketObjects returns a list of an AWS s3 bucket's objects.
func listBucketObjects(svc s3iface.S3API) (*s3.ListObjectsV2Output, error) {
	resp, err := svc.ListObjectsV2(&s3.ListObjectsV2Input{Bucket: aws.String(appSecrets.BucketName)})
	if err != nil {
		fmt.Printf("Unable to list items in bucket %q", appSecrets.BucketName)
		return &s3.ListObjectsV2Output{}, err
	}

	return resp, nil
}

// DownloadSignatures compares signature databases on disk with those in the clam mirror bucket.
// It will download copies of the databases if found to be newer than what's on disk.
func DownloadSignatures(svc s3iface.S3API, resp *s3.ListObjectsV2Output) error {
	downloader := s3manager.NewDownloaderWithClient(svc)

	// Loop through bucket contents, and compare with our set. If file is a match, then download it.
	for _, item := range resp.Contents {
		if strings.HasSuffix(*item.Key, ".gz") {
			splitItem := strings.Split(*item.Key, ".gz")
			baseItem := splitItem[0]

			newFile, err := os.Create(baseItem)
			if err != nil {
				fmt.Println("Unable to open file:", item)
				return err
			}
			//defer newFile.Close()

			buf := aws.NewWriteAtBuffer([]byte{})

			requestInput := s3.GetObjectInput{
				Bucket: aws.String("osd-fr-signature-mirror"),
				Key:    aws.String(*item.Key),
			}

			_, err = downloader.Download(buf, &requestInput)
			if err != nil {
				fmt.Println("Unable to download item:", item)
				return fmt.Errorf("Download failed: %v\n", err)
			}

			r := bytes.NewReader(buf.Bytes())

			zr, err := gzip.NewReader(r)
			if err != nil {
				fmt.Println(err)
			}
			//defer zr.Close()

			sha := sha256.New()
			data := io.TeeReader(zr, sha)
			checksum := hex.EncodeToString(sha.Sum(nil))

			if _, err := io.Copy(newFile, data); err != nil {
				return fmt.Errorf("could not copy zr to newFile: %v\n", err)
			}

			if err := zr.Close(); err != nil {
				return fmt.Errorf("could not close zr: %v\n", err)
			}

			if err := newFile.Close(); err != nil {
				return fmt.Errorf("could not close newFile: %v\n", err)
			}

			println("checksum is: ", checksum)

			if lf, ok := checksumCache.LocalFiles[f]; ok {
				lf.Name = f
				lf.Checksum = hex.EncodeToString(sha.Sum(nil))

				checksumCache.LocalFiles[f] = lf
			}

			fmt.Println("Downloaded the following:")
			fmt.Println("Name:         ", *item.Key)
			fmt.Println("Last modified:", *item.LastModified)
			fmt.Println("Size:         ", *item.Size, "bytes")
			fmt.Println("")
		}
	}
	return nil
}

// DownloadSignatures compares signature databases on disk with those in the clam mirror bucket.
// It will download copies of the databases if found to be newer than what's on disk.
/*
func DownloadSignatures(svc s3iface.S3API, resp *s3.ListObjectsV2Output) error {
	downloader := s3manager.NewDownloaderWithClient(svc)

	// Loop through bucket contents, and compare with our set. If file is a match, then download it.
	for _, item := range resp.Contents {
		splitItem := strings.Split(*item.Key, ".")
		baseItem := splitItem[0]

		if _, ok := checksumCache[splitItem[0]]; ok {
			newFile, err := os.Create(filepath.Join(config.ClamInstallDir, item))
			if err != nil {
				fmt.Println("Unable to open file:", item)
				return err
			}
			defer newFile.Close()

			buf := aws.NewWriteAtBuffer([]byte{})
			zr, err := gzip.NewReader(&buf)
			if err != nil {
				fmt.Println(err)
			}

			//sha := sha256.New()
			//data := io.TeeReader(newFile, sha)

			_, err = downloader.Download(buf,
				&s3.GetObjectInput{
					Bucket: aws.String(configFile.ClamMirrorBucket),
					Key:    aws.String(*item.Key),
				})
			if err != nil {
				fmt.Println("Unable to download item:", item)
				return err
			}

			fmt.Println("Downloaded the following:")
			fmt.Println("Name:         ", *item.Key)
			fmt.Println("Last modified:", *item.LastModified)
			fmt.Println("Size:         ", *item.Size, "bytes")
			fmt.Println("")
		}
	}
	return nil
}

// DownloadSignatures compares signature databases on disk with those in the clam mirror bucket.
// It will download copies of the databases if found to be newer than what's on disk.
func DownloadSignatures(svc s3iface.S3API, resp *s3.ListObjectsV2Output) error {
	downloader := s3manager.NewDownloaderWithClient(svc)

	// Loop through bucket contents, and compare with our json array. If file is a match, then
	// check if doesn't exist, and check if the bucket's file is newer. Download it in those cases.
	for _, item := range resp.Contents {
		for _, localItem := range appSecrets.ConfigFiles {
			if *item.Key == localItem {
				fileStat, err := os.Stat(filepath.Join(config.ClamInstallDir, localItem))
				if os.IsNotExist(err) || fileStat.ModTime().Before(*item.LastModified) {

					newFile, err := os.Create(filepath.Join(config.ClamInstallDir, localItem))
					if err != nil {
						fmt.Println("Unable to open file:", item)
						return err
					}

					defer newFile.Close()

					_, err = downloader.Download(newFile,
						&s3.GetObjectInput{
							Bucket: aws.String(configFile.ClamMirrorBucket),
							Key:    aws.String(*item.Key),
						})
					if err != nil {
						fmt.Println("Unable to download item:", item)
						return err
					}

					fmt.Println("Downloaded the following:")
					fmt.Println("Name:         ", *item.Key)
					fmt.Println("Last modified:", *item.LastModified)
					fmt.Println("Size:         ", *item.Size, "bytes")
					fmt.Println("")

				} else if err != nil {
					fmt.Println("Hit an issue opening the file:")
					return err
				}
			}
		}
	}
	return nil
}




func bucketDownload(bucket, fileDir string, fileList []string) error {
	sess, err := getSession()
	if err != nil {
		return fmt.Errorf("Error returned from getSession: %v\n", err)
	}

	svc := getService(sess)

	downloader := s3manager.NewDownloaderWithClient(svc)
	return nil
}
*/

// getService returns a new S3 client service from an existing session.
func getService(sess *session.Session) *s3.S3 {
	svc := s3.New(sess)

	return svc
}

func downloadSingle(item string, svc *s3.S3) error {
	buf := aws.NewWriteAtBuffer([]byte{})

	downloader := s3manager.NewDownloaderWithClient(svc)

	requestInput := s3.GetObjectInput{
		Bucket: aws.String("osd-fr-signature-mirror"),
		Key:    aws.String(item),
	}

	_, err := downloader.Download(buf, &requestInput)
	if err != nil {
		fmt.Println("Unable to download item:", item)
		return fmt.Errorf("Download failed: %v\n", err)
	}

	err = json.Unmarshal(buf.Bytes(), &checksumCache)
	if err != nil {
		return fmt.Errorf("Failed to Unmarshal bytes to checksumCache: %v\n", err)
	}
	return nil
}

func main() {
	fmt.Println("signature-updater v0.0.6")

	filePath := os.Getenv("CLAM_UPDATE_SECRETS_FILE")

	err := loadConfigFile(filePath, &appSecrets)
	if err != nil {
		fmt.Println("Error reading file: ", err)
	}

	if appSecrets.ClamConfigDir == "" {
		appSecrets.ClamConfigDir = os.Getenv("CLAM_DB_DIRECTORY")

		if appSecrets.ClamConfigDir == "" {
			appSecrets.ClamConfigDir = "/var/lib/clamav/"
		}
	}

	// be tolerant of an env var path not already suffixed with a trailing slash
	if !strings.HasSuffix(appSecrets.ClamConfigDir, "/") {
		appSecrets.ClamConfigDir = appSecrets.ClamConfigDir + "/"
	}

	for _, item := range appSecrets.ConfigFiles {
		appSecrets.ConfigFileMap[item] = struct{}{}
	}

	confLen := len(appSecrets.ConfigFiles)
	checksumCache = models.NewVerifiedFiles(confLen)

	sess, err := getSession()
	if err != nil {
		fmt.Println("Error returned from GetSession:", err)
	}

	svc := getService(sess)

	err = downloadSingle(appSecrets.ChecksumFile, svc)
	if err != nil {
		fmt.Println("Error returned from downloadSingle for ChecksumFile file:", err)
	}

	resp, err := listBucketObjects(svc)
	if err != nil {
		fmt.Println("Error returned from ListBucketObjects:", err)
	}

	err = DownloadSignatures(svc, resp)
	if err != nil {
		fmt.Println("Error returned from DownloadSignatures:", err)
	}

	/*
		sess, err := getSession()
		if err != nil {
			fmt.Println("Error returned from getSession: ", err)
		}

		svc := getService(sess)

		err = DownloadSignatures(svc)
		if err != nil {
			fmt.Println("Error uploading files to bucket: ", err)
		}
	*/

	err = upload(appSecrets.BucketName, appSecrets.ClamConfigDir, appSecrets.ConfigFiles)
	if err != nil {
		fmt.Println("Error uploading files to bucket: ", err)
	}

	err = uploadSingle(appSecrets.BucketName, appSecrets.ChecksumFile, checksumCache)
	if err != nil {
		fmt.Println("Error uploading verifiedFiles checksum map to bucket: ", err)
	}
}

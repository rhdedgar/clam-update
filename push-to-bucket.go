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
	"os/exec"
	"path"
	"strings"
	"time"

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
	//checksumCache = &models.VerifiedFiles{}
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

// gzipFile reads a filename and returns a pointer to a bytes.Buffer containing the gzipped file.
func gzipFile(fileName string) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	fileBytes, err := ioutil.ReadFile(fileName)
	if err != nil {
		return &buf, fmt.Errorf("error ioutil reading file: %v\n", err)
	}

	zw := gzip.NewWriter(&buf)
	zw.ModTime = time.Now()

	if _, err = zw.Write(fileBytes); err != nil {
		return &buf, fmt.Errorf("error writing zw: %v\n", err)
	}

	if err := zw.Close(); err != nil {
		return &buf, fmt.Errorf("error closing zw: %v\n", err)
	}

	return &buf, nil
}

// gzipFile2 reads a filename and returns a pointer to a bytes.Buffer containing the gzipped file.
func gzipFile2(fileBytes []byte) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	//fileBytes, err := ioutil.ReadFile(fileName)
	//if err != nil {
	//	return &buf, fmt.Errorf("error ioutil reading file: %v\n", err)
	//}

	zw := gzip.NewWriter(&buf)
	zw.ModTime = time.Now()

	if _, err := zw.Write(fileBytes); err != nil {
		return &buf, fmt.Errorf("error writing zw: %v\n", err)
	}

	if err := zw.Close(); err != nil {
		return &buf, fmt.Errorf("error closing zw: %v\n", err)
	}

	return &buf, nil
}

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

		tr := io.TeeReader(file, sha)
		checksum := hex.EncodeToString(sha.Sum(nil))

		if checksum == "" {
			continue
		}

		err = uploadSingle(bucket, fileName+"_checksum.txt", checksum)
		if err != nil {
			return fmt.Errorf("Unable to upload %v to %v, %v\n", filePath, bucket, err)
		}
		// END HASH

		//reader := bufio.NewReader(file)
		//zipData, err := gzipFile(filePath)
		//r := bytes.NewReader(data)
		fileBytes, err := io.ReadAll(tr)
		if err != nil {
			fmt.Println(err)
		}

		zipData, err := gzipFile2(fileBytes)
		if err != nil {
			fmt.Printf("Unable to open file %v: %v\n", filePath, err)
			continue
		}

		//zipData, err := gzipFile(data)
		//zw := gzip.NewWriter(file)
		//zw.ModTime = time.Now()

		err = uploadSingle(bucket, fileName+".gz", zipData)
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

			//sha := sha256.New()
			//data := io.TeeReader(zr, sha)
			//checksum := hex.EncodeToString(sha.Sum(nil))

			if _, err := io.Copy(newFile, zr); err != nil {
				return fmt.Errorf("could not copy zr to newFile: %v\n", err)
			}

			if err := zr.Close(); err != nil {
				return fmt.Errorf("could not close zr: %v\n", err)
			}

			if err := newFile.Close(); err != nil {
				return fmt.Errorf("could not close newFile: %v\n", err)
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

	/*
		err = json.Unmarshal(buf.Bytes(), &checksumCache)
		if err != nil {
			return fmt.Errorf("Failed to Unmarshal bytes to checksumCache: %v\n", err)
		}
	*/
	return nil
}

func runScripts(scripts ...string) error {
	for _, script := range scripts {
		cmd := exec.Command(script)
		err := cmd.Run()
		if err != nil {
			return fmt.Errorf("error running script: %v\n", err)
		}
	}
	return nil
}

func main() {
	fmt.Println("signature-updater v0.0.7")

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

	//confLen := len(appSecrets.ConfigFiles)
	//checksumCache = models.NewVerifiedFiles(confLen)

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

	err = runScripts(
		"/usr/bin/freshclam",
		"/usr/sbin/clamav-unofficial-sigs.sh",
		//"/usr/local/bin/pull-custom-signatures.sh",
	)
	if err != nil {
		fmt.Println("Error running scripts: ", err)
	}

	cloneURL := fmt.Sprintf("https://oauth2:" + appSecrets.GitPullToken + "@gitlab.cee.redhat.com/service/clamav-custom-signatures.git")
	cmd := exec.Command("git", "clone", cloneURL)
	err = cmd.Run()
	if err != nil {
		fmt.Printf("error running git clone: %v\n", err)
	}

	cmd = exec.Command("cp", "clamav-custom-signatures/*", appSecrets.ClamConfigDir)
	err = cmd.Run()
	if err != nil {
		fmt.Printf("error copying custom signature files to clam dir: %v\n", err)
	}

	err = upload(appSecrets.BucketName, appSecrets.ClamConfigDir, appSecrets.ConfigFiles)
	if err != nil {
		fmt.Println("Error uploading files to bucket: ", err)
	}
	fmt.Println("Finished running. Exiting.")
}

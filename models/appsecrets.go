package models

// AppSecrets holds needed config info for the application to function
type AppSecrets struct {
	OcavOpsFiles      []string `json:"ocav_ops_files"`
	OcavTimestampPath string   `json:"ocav_timestamp_path"`
	OcavCredsFile     string   `json:"ocav_creds_file"`
	OcavS3Bucket      string   `json:"ocav_s3_bucket"`
}

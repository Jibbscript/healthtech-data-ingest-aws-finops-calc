package domain

import "time"

type CaptureStatus string

const (
	CapturePending   CaptureStatus = "pending"
	CaptureSucceeded CaptureStatus = "succeeded"
	CaptureFailed    CaptureStatus = "failed"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
}

type Device struct {
	ID                    string    `json:"id"`
	UserID                string    `json:"user_id"`
	CertificateThumbprint string    `json:"certificate_thumbprint"`
	EnrolledAt            time.Time `json:"enrolled_at"`
}

type Capture struct {
	ID          string        `json:"id"`
	UserID      string        `json:"user_id"`
	DeviceID    string        `json:"device_id"`
	S3Key       string        `json:"s3_key"`
	ContentHash string        `json:"content_hash"`
	SizeBytes   int64         `json:"size_bytes"`
	Status      CaptureStatus `json:"status"`
	CapturedAt  time.Time     `json:"captured_at"`
	Failure     string        `json:"failure,omitempty"`
}

type Finding struct {
	ID         string    `json:"id"`
	CaptureID  string    `json:"capture_id"`
	UserID     string    `json:"user_id"`
	Result     string    `json:"result"`
	Confidence float64   `json:"confidence"`
	Reason     string    `json:"reason,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type IngestJob struct {
	CaptureID   string    `json:"capture_id"`
	DeviceID    string    `json:"device_id"`
	UserID      string    `json:"user_id"`
	S3Key       string    `json:"s3_key"`
	ContentHash string    `json:"content_hash"`
	CapturedAt  time.Time `json:"captured_at"`
	Traceparent string    `json:"traceparent,omitempty"`
}

type InferRequest struct {
	CaptureID    string `json:"capture_id"`
	S3Key        string `json:"s3_key"`
	ModelVersion string `json:"model_version"`
	ContentHash  string `json:"content_hash"`
}

type Pricing struct {
	Region               string  `json:"region"`
	Source               string  `json:"source"`
	FargateVCPUPerHour   float64 `json:"fargate_vcpu_per_hour,omitempty"`
	FargateGBPerHour     float64 `json:"fargate_gb_per_hour,omitempty"`
	S3GBMonth            float64 `json:"s3_gb_month,omitempty"`
	S3Class              string  `json:"s3_class,omitempty"`
	SQSRequestPerMillion float64 `json:"sqs_request_per_million,omitempty"`
	RDSInstancePerHour   float64 `json:"rds_instance_per_hour,omitempty"`
	DataTransferGB       float64 `json:"data_transfer_gb,omitempty"`
	Currency             string  `json:"currency"`
}

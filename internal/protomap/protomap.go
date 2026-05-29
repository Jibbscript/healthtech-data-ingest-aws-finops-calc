// Package protomap converts domain types to their generated protobuf
// equivalents, shared by the ingest and api services so the mapping lives
// in exactly one place.
package protomap

import (
	"github.com/jibbscript/throne-backend-poc/internal/domain"
	thronev1 "github.com/jibbscript/throne-backend-poc/internal/gen/throne/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func CaptureToProto(c domain.Capture) *thronev1.Capture {
	return &thronev1.Capture{
		Id:          c.ID,
		UserId:      c.UserID,
		DeviceId:    c.DeviceID,
		S3Key:       c.S3Key,
		ContentHash: c.ContentHash,
		SizeBytes:   c.SizeBytes,
		Status:      string(c.Status),
		CapturedAt:  timestamppb.New(c.CapturedAt),
	}
}

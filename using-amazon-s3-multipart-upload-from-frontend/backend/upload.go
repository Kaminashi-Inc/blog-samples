package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	v4 "github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MultipartUploadHandler struct {
	client *s3.Client
}

func newMultipartUploadHandler(ctx context.Context) (MultipartUploadHandler, error) {
	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return MultipartUploadHandler{}, err
	}
	client := s3.NewFromConfig(cfg)
	return MultipartUploadHandler{
		client: client,
	}, nil
}

type MultipartUploadTarget struct {
	UploadID string `json:"uploadId"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
}

type StartMultipartUploadRequest struct {
	PartCount int `json:"partCount"`
}

type StartMultipartUploadResponse struct {
	MultipartUploadTarget MultipartUploadTarget `json:"multipartUploadTarget"`
	Region                string                `json:"region"`
	UploadPartURLInfos    []UploadPartURLInfo   `json:"uploadPartURLInfos"`
}

type UploadPartURLInfo struct {
	PartNumber    int32  `json:"partNumber"`
	UploadPartURL string `json:"uploadPartURL"`
}

func handleStartMultipartUpload(c *gin.Context) {

	var req StartMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusBadRequest)
	}

	h, err := newMultipartUploadHandler(c.Request.Context())
	if err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	resp, err := h.startMultiPartUpload(c.Request.Context(), req)
	if err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h MultipartUploadHandler) startMultiPartUpload(ctx context.Context, req StartMultipartUploadRequest) (StartMultipartUploadResponse, error) {
	multipartUploadTarget, err := h.createMultipartUpload(ctx)
	if err != nil {
		return StartMultipartUploadResponse{}, err
	}

	uploadPartURLInfos, err := h.createUploadPartURLInfos(ctx, multipartUploadTarget, req.PartCount)
	if err != nil {
		return StartMultipartUploadResponse{}, err
	}

	region := os.Getenv("AWS_REGION")

	return StartMultipartUploadResponse{
		MultipartUploadTarget: multipartUploadTarget,
		Region:                region,
		UploadPartURLInfos:    uploadPartURLInfos,
	}, nil
}

func (h MultipartUploadHandler) createMultipartUpload(ctx context.Context) (MultipartUploadTarget, error) {

	key := uuid.NewString()
	bucketName := os.Getenv("AWS_BUCKET")
	resp, err := h.client.CreateMultipartUpload(ctx, &s3.CreateMultipartUploadInput{
		Bucket: &bucketName,
		Key:    &key,
	})

	if err != nil {
		return MultipartUploadTarget{}, err
	}

	return MultipartUploadTarget{
		UploadID: *resp.UploadId,
		Bucket:   bucketName,
		Key:      key,
	}, nil
}

func (h MultipartUploadHandler) createUploadPartURLInfos(ctx context.Context, multipartUploadTarget MultipartUploadTarget, partCount int) ([]UploadPartURLInfo, error) {
	uploadPartURLInfos := make([]UploadPartURLInfo, 0, partCount)
	for i := 0; i < partCount; i++ {
		partNumber := int32(i + 1)
		uploadPartURL, err := h.createPresignedURL(ctx, multipartUploadTarget, partNumber)
		if err != nil {
			return nil, err
		}
		uploadPartURLInfos = append(uploadPartURLInfos, UploadPartURLInfo{
			PartNumber:    partNumber,
			UploadPartURL: uploadPartURL,
		})
	}
	return uploadPartURLInfos, nil
}

func (h MultipartUploadHandler) createPresignedURL(ctx context.Context, multipartUploadTarget MultipartUploadTarget, partNumber int32) (string, error) {

	resp, err := s3.NewPresignClient(h.client).PresignPutObject(ctx, &s3.PutObjectInput{
		Bucket: &multipartUploadTarget.Bucket,
		Key:    &multipartUploadTarget.Key,
	}, func(options *s3.PresignOptions) {
		options.Expires = time.Duration(60 * time.Second)
		originalPresigner := options.Presigner
		options.Presigner = uploadPartPresigner{
			uploadId:   multipartUploadTarget.UploadID,
			partNumber: partNumber,
			orig:       originalPresigner,
		}
	})

	if err != nil {
		return "", err
	}

	return resp.URL, nil
}

type uploadPartPresigner struct {
	uploadId   string
	partNumber int32
	orig       s3.HTTPPresignerV4
}

func (pw uploadPartPresigner) PresignHTTP(
	ctx context.Context, credentials aws.Credentials, r *http.Request,
	payloadHash string, service string, region string, signingTime time.Time,
	optFns ...func(*v4.SignerOptions),
) (url string, signedHeader http.Header, err error) {
	q := r.URL.Query()
	q.Add("uploadId", pw.uploadId)
	q.Add("partNumber", fmt.Sprintf("%d", pw.partNumber))
	r.URL.RawQuery = q.Encode()
	return pw.orig.PresignHTTP(ctx, credentials, r, payloadHash, service, region, signingTime, optFns...)
}

type UploadedPartInfo struct {
	ETag       string `json:"eTag"`
	PartNumber int32  `json:"partNumber"`
}

type CompleteMultipartUploadRequest struct {
	UploadID string             `json:"uploadId"`
	Bucket   string             `json:"bucket"`
	Key      string             `json:"key"`
	Parts    []UploadedPartInfo `json:"parts"`
	Checksum string             `json:"checksum"`
}

func handleCompleteMultipartUpload(c *gin.Context) {

	var req CompleteMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	h, err := newMultipartUploadHandler(c.Request.Context())
	if err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := h.completeMultipartUpload(c.Request.Context(), req); err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (h MultipartUploadHandler) completeMultipartUpload(ctx context.Context, req CompleteMultipartUploadRequest) error {

	completeParts := make([]types.CompletedPart, 0, len(req.Parts))
	for _, p := range req.Parts {
		completeParts = append(completeParts, types.CompletedPart{
			ETag:       &p.ETag,
			PartNumber: &p.PartNumber,
		})
	}

	resp, err := h.client.CompleteMultipartUpload(ctx, &s3.CompleteMultipartUploadInput{
		Bucket:   &req.Bucket,
		Key:      &req.Key,
		UploadId: &req.UploadID,
		MultipartUpload: &types.CompletedMultipartUpload{
			Parts: completeParts,
		},
	})

	if err != nil {
		return err

	}

	expectedETag := fmt.Sprintf("\"%s-%d\"", req.Checksum, len(req.Parts))
	if *resp.ETag != expectedETag {
		slog.Warn(fmt.Sprintf("ETag mismatch, expected %s, got %s", expectedETag, *resp.ETag))
	}

	return err
}

type AbortMultipartUploadRequest struct {
	UploadID string `json:"uploadId"`
	Bucket   string `json:"bucket"`
	Key      string `json:"key"`
}

func handleAbortMultipartUpload(c *gin.Context) {

	var req AbortMultipartUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusBadRequest)
		return
	}

	h, err := newMultipartUploadHandler(c.Request.Context())
	if err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	if err := h.abortMultipartUpload(c.Request.Context(), req); err != nil {
		slog.Error(err.Error())
		c.AbortWithStatus(http.StatusInternalServerError)
	}
}

func (h MultipartUploadHandler) abortMultipartUpload(ctx context.Context, req AbortMultipartUploadRequest) error {
	_, err := h.client.AbortMultipartUpload(ctx, &s3.AbortMultipartUploadInput{
		Bucket:   &req.Bucket,
		Key:      &req.Key,
		UploadId: &req.UploadID,
	})
	return err
}

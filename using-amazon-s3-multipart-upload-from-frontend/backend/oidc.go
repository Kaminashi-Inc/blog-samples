package main

import (
	"context"
	"net/http"
	"os"

	"log/slog"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/cognitoidentity"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type OpenIDTokenHandler struct {
	client *cognitoidentity.Client
}

func NewOpenIDTokenHandler(ctx context.Context) (OpenIDTokenHandler, error) {

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return OpenIDTokenHandler{}, err
	}
	client := cognitoidentity.NewFromConfig(cfg)
	return OpenIDTokenHandler{
		client: client,
	}, nil

}

type OpenIDToken struct {
	IdentityID string `json:"identityId"`
	Token      string `json:"token"`
}

type GetOpenIDTokenResponse struct {
	OpenIDToken OpenIDToken `json:"openIDToken"`
	Region      string      `json:"region"`
	Bucket      string      `json:"bucket"`
	KeyPrefix   string      `json:"keyPrefix"`
}

func handleGetOpenIDToken(ginCtx *gin.Context) {
	ctx := ginCtx.Request.Context()

	h, err := NewOpenIDTokenHandler(ctx)
	if err != nil {
		slog.Error(err.Error())
		ginCtx.AbortWithStatus(http.StatusInternalServerError)
		return
	}

	ret, err := h.getOpenIDToken(ctx)
	if err != nil {
		slog.Error(err.Error())
		ginCtx.AbortWithStatus(http.StatusInternalServerError)
		return
	}
	ginCtx.JSON(http.StatusOK, ret)
}

func (h OpenIDTokenHandler) getOpenIDToken(ctx context.Context) (GetOpenIDTokenResponse, error) {

	identityPoolID := os.Getenv("AWS_IDENTITY_POOL_ID")
	loginProvider := os.Getenv("AWS_LOGIN_PROVIDER")
	loginName := getLoginName(ctx)

	resp, err := h.client.GetOpenIdTokenForDeveloperIdentity(ctx, &cognitoidentity.GetOpenIdTokenForDeveloperIdentityInput{
		IdentityPoolId: aws.String(identityPoolID),
		Logins: map[string]string{
			loginProvider: loginName,
		},
		TokenDuration: aws.Int64(15 * 60),
		PrincipalTags: map[string]string{
			"loginName": loginName,
		},
	})

	if err != nil {
		return GetOpenIDTokenResponse{}, err
	}

	bucketName := os.Getenv("AWS_BUCKET")
	region := os.Getenv("AWS_REGION")

	return GetOpenIDTokenResponse{
		OpenIDToken: OpenIDToken{
			IdentityID: *resp.IdentityId,
			Token:      *resp.Token,
		},
		Region:    region,
		Bucket:    bucketName,
		KeyPrefix: loginName,
	}, nil
}

func getLoginName(ctx context.Context) string {
	// 実際のアプリケーションでは、セッション情報などから、ログイン中のユーザーの識別子を取得します。
	return "user_" + uuid.NewString()
}

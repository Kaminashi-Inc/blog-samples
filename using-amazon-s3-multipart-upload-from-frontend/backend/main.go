package main

import (
	"log"
	"net/http"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		panic(err)
	}
	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins: []string{"http://localhost:51703"}, // フロントエンドアプリケーションのオリジンを指定する
		AllowMethods: []string{"GET", "POST"},
		AllowHeaders: []string{"Content-Type"},
	}))
	// 👇 方法1 で利用するエンドポイント
	router.GET("/openIDToken", handleGetOpenIDToken)

	// 👇 方法2 で利用するエンドポイント
	router.POST("/startMultipartUpload", handleStartMultipartUpload)
	router.POST("/completeMultipartUpload", handleCompleteMultipartUpload)
	router.POST("/abortMultipartUpload", handleAbortMultipartUpload)

	log.Fatalln(http.ListenAndServe(":8080", router))
}

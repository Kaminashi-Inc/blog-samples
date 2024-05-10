#########################
# S3 Bucket
#########################

resource "aws_s3_bucket" "s3_bucket" {
  bucket = var.s3_bucket_name
}

resource "aws_s3_bucket_cors_configuration" "s3_bucket_cors_configuration" {
  bucket = aws_s3_bucket.s3_bucket.bucket
  cors_rule {
    allowed_headers = ["*"]
    allowed_methods = [
      "PUT",
      "POST",   # CreateMultipartUpload をフロントエンドから実行するときに必要
      "DELETE", # AbortMultipartUpload をフロントエンドから実行するときに必要
    ]
    allowed_origins = ["http://localhost:51703"] # フロントエンドアプリケーションのオリジンを指定する
    expose_headers  = ["ETag"]
    max_age_seconds = 3000
  }
}

# 7 日間経過後、不完全なマルチパートアップロードを削除するライフサイクルルール
resource "aws_s3_bucket_lifecycle_configuration" "s3_bucket_lifecycle_configuration" {
  bucket = aws_s3_bucket.s3_bucket.id
  rule {
    id     = "abort-incomplete-multipart-upload-after-7-days"
    status = "Enabled"
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

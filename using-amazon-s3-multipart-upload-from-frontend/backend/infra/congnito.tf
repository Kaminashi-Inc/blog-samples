

#########################
# Cognito Identity Pool
#########################

resource "aws_cognito_identity_pool" "cognito_identity_pool" {
  identity_pool_name               = "my-cognito-identity-pool-name"
  allow_unauthenticated_identities = false

  developer_provider_name = var.developer_provider_name
}

data "aws_iam_policy_document" "s3_policy_document" {
  statement {
    effect = "Allow"
    actions = [
      "s3:PutObject"
    ]
    resources = [
      # PrincipalTag を使用して、ログインユーザーごとにプレフィックスを切り分けています
      "arn:aws:s3:::${aws_s3_bucket.s3_bucket.bucket}/$${aws:PrincipalTag/loginName}/*"
    ]
  }
}

resource "aws_iam_policy" "s3_policy" {
  name        = "s3_policy"
  description = "IAM policy allowing s3:PutObject to the s3_bucket"
  policy      = data.aws_iam_policy_document.s3_policy_document.json
}

data "aws_iam_policy_document" "cognito_trust_relationship_policy_document" {
  statement {
    effect = "Allow"
    principals {
      type        = "Federated"
      identifiers = ["cognito-identity.amazonaws.com"]
    }
    actions = [
      "sts:AssumeRoleWithWebIdentity",
      "sts:TagSession" # PrincipalTag を使用しない場合は不要です
    ]
    condition {
      test     = "StringEquals"
      variable = "cognito-identity.amazonaws.com:aud"
      values = [
        "${aws_cognito_identity_pool.cognito_identity_pool.id}"
      ]
    }
    condition {
      test     = "ForAnyValue:StringLike"
      variable = "cognito-identity.amazonaws.com:amr"
      values = [
        "authenticated"
      ]
    }
  }
}

resource "aws_iam_role" "cognito_role" {
  name               = "cognito-authenticated-user-role"
  assume_role_policy = data.aws_iam_policy_document.cognito_trust_relationship_policy_document.json
}

resource "aws_iam_role_policy_attachment" "s3_policy_attachment" {
  role       = aws_iam_role.cognito_role.name
  policy_arn = aws_iam_policy.s3_policy.arn
}

resource "aws_cognito_identity_pool_roles_attachment" "cognito_identity_pool_roles_attachment" {
  identity_pool_id = aws_cognito_identity_pool.cognito_identity_pool.id
  roles = {
    "authenticated" = aws_iam_role.cognito_role.arn
  }
}

output "cognito_identity_pool_id" {
  value = aws_cognito_identity_pool.cognito_identity_pool.id
}

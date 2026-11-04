# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# One encrypted, versioned, private bucket that logs its own access under access-logs/
# and replicates the state under preview/ to a bucket in another region.

terraform {
  required_providers {
    aws = {
      source = "hashicorp/aws"
    }
  }
}

variable "name" {
  type = string
}

variable "replication_role_arn" {
  type = string
}

variable "destination_arn" {
  type = string
}

variable "destination_key_arn" {
  type = string
}

locals {
  account = data.aws_caller_identity.current.account_id
}

data "aws_caller_identity" "current" {}

resource "aws_kms_key" "this" {
  description         = "Encrypts ${var.name}"
  enable_key_rotation = true
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "Administer"
        Effect    = "Allow"
        Principal = { AWS = "arn:aws:iam::${local.account}:root" }
        Action    = "kms:*"
        Resource  = "*"
      },
      # An AWS managed key cannot encrypt access logs.
      {
        Sid       = "DeliverAccessLogs"
        Effect    = "Allow"
        Principal = { Service = "logging.s3.amazonaws.com" }
        Action    = "kms:GenerateDataKey*"
        Resource  = "*"
      },
    ]
  })
}

resource "aws_s3_bucket" "this" {
  bucket = var.name
}

resource "aws_s3_bucket_versioning" "this" {
  bucket = aws_s3_bucket.this.id
  versioning_configuration {
    status = "Enabled"
  }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "this" {
  bucket = aws_s3_bucket.this.id
  rule {
    bucket_key_enabled = true
    apply_server_side_encryption_by_default {
      sse_algorithm     = "aws:kms"
      kms_master_key_id = aws_kms_key.this.arn
    }
  }
}

resource "aws_s3_bucket_public_access_block" "this" {
  bucket                  = aws_s3_bucket.this.id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

resource "aws_s3_bucket_policy" "this" {
  bucket     = aws_s3_bucket.this.id
  depends_on = [aws_s3_bucket_public_access_block.this]
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid       = "DeliverAccessLogs"
      Effect    = "Allow"
      Principal = { Service = "logging.s3.amazonaws.com" }
      Action    = "s3:PutObject"
      Resource  = "${aws_s3_bucket.this.arn}/access-logs/*"
      Condition = {
        ArnLike      = { "aws:SourceArn" = aws_s3_bucket.this.arn }
        StringEquals = { "aws:SourceAccount" = local.account }
      }
    }]
  })
}

resource "aws_s3_bucket_logging" "this" {
  bucket        = aws_s3_bucket.this.id
  target_bucket = aws_s3_bucket.this.id
  target_prefix = "access-logs/"
}

resource "aws_s3_bucket_lifecycle_configuration" "this" {
  bucket     = aws_s3_bucket.this.id
  depends_on = [aws_s3_bucket_versioning.this]
  rule {
    id     = "state"
    status = "Enabled"
    filter {}
    noncurrent_version_expiration {
      noncurrent_days = 90
    }
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
  rule {
    id     = "access-logs"
    status = "Enabled"
    filter {
      prefix = "access-logs/"
    }
    expiration {
      days = 90
    }
  }
}

resource "aws_s3_bucket_notification" "this" {
  bucket      = aws_s3_bucket.this.id
  eventbridge = true
}

resource "aws_s3_bucket_replication_configuration" "this" {
  bucket     = aws_s3_bucket.this.id
  role       = var.replication_role_arn
  depends_on = [aws_s3_bucket_versioning.this]
  rule {
    id     = "state"
    status = "Enabled"
    filter {
      prefix = "preview/"
    }
    delete_marker_replication {
      status = "Enabled"
    }
    source_selection_criteria {
      sse_kms_encrypted_objects {
        status = "Enabled"
      }
    }
    destination {
      bucket = var.destination_arn
      encryption_configuration {
        replica_kms_key_id = var.destination_key_arn
      }
    }
  }
}

output "id" {
  value = aws_s3_bucket.this.id
}

output "arn" {
  value = aws_s3_bucket.this.arn
}

output "key_arn" {
  value = aws_kms_key.this.arn
}

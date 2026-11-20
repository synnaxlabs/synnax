# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# The S3 bucket that holds the preview host's Terraform state. Apply this once before
# deploy/preview. Its own state stays local: losing it costs one terraform import.

terraform {
  required_version = ">= 1.10"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = "us-east-1"
  default_tags {
    tags = { project = "synnax-agents", owner = "preview" }
  }
}

provider "aws" {
  alias  = "replica"
  region = "us-west-2"
  default_tags {
    tags = { project = "synnax-agents", owner = "preview" }
  }
}

data "aws_caller_identity" "current" {}

locals {
  bucket = "synnax-preview-terraform-${data.aws_caller_identity.current.account_id}"
}

module "state" {
  source               = "./bucket"
  name                 = local.bucket
  replication_role_arn = aws_iam_role.replication.arn
  destination_arn      = module.replica.arn
  destination_key_arn  = module.replica.key_arn
}

# Replicates back, so a write to the replica during a regional outage is not stranded.
module "replica" {
  source               = "./bucket"
  name                 = "${local.bucket}-replica"
  replication_role_arn = aws_iam_role.replication.arn
  destination_arn      = module.state.arn
  destination_key_arn  = module.state.key_arn
  providers            = { aws = aws.replica }
}

data "aws_iam_policy_document" "replication_trust" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["s3.amazonaws.com"]
    }
  }
}

data "aws_iam_policy_document" "replication" {
  statement {
    actions   = ["s3:GetReplicationConfiguration", "s3:ListBucket"]
    resources = [module.state.arn, module.replica.arn]
  }
  statement {
    actions = [
      "s3:GetObjectVersionAcl",
      "s3:GetObjectVersionForReplication",
      "s3:GetObjectVersionTagging",
      "s3:ReplicateDelete",
      "s3:ReplicateObject",
      "s3:ReplicateTags",
    ]
    resources = ["${module.state.arn}/*", "${module.replica.arn}/*"]
  }
  statement {
    actions   = ["kms:Decrypt", "kms:Encrypt", "kms:GenerateDataKey"]
    resources = [module.state.key_arn, module.replica.key_arn]
  }
}

resource "aws_iam_role" "replication" {
  name               = "synnax-preview-state-replication"
  assume_role_policy = data.aws_iam_policy_document.replication_trust.json
}

resource "aws_iam_role_policy" "replication" {
  name   = "replication"
  role   = aws_iam_role.replication.id
  policy = data.aws_iam_policy_document.replication.json
}

moved {
  from = aws_s3_bucket.state
  to   = module.state.aws_s3_bucket.this
}

moved {
  from = aws_s3_bucket_versioning.state
  to   = module.state.aws_s3_bucket_versioning.this
}

moved {
  from = aws_s3_bucket_server_side_encryption_configuration.state
  to   = module.state.aws_s3_bucket_server_side_encryption_configuration.this
}

moved {
  from = aws_s3_bucket_public_access_block.state
  to   = module.state.aws_s3_bucket_public_access_block.this
}

output "bucket" {
  value = module.state.id
}

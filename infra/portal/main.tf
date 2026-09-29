# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

terraform {
  required_version = ">= 1.9"

  cloud {
    organization = "synnaxlabs"
    workspaces {
      name = "hub"
    }
  }

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
    vercel = {
      source  = "vercel/vercel"
      version = "~> 4.0"
    }
    github = {
      source  = "integrations/github"
      version = "~> 6.0"
    }
    random = {
      source  = "hashicorp/random"
      version = "~> 3.6"
    }
  }
}

provider "aws" {
  region = var.aws_region
}

provider "vercel" {
  team = var.vercel_team_id
}

provider "github" {
  owner = "synnaxlabs"
}

# The signing key. ML-DSA-44, sign-only, and never exportable: every license key a Core
# accepts is signed under it, and its public half is compiled into the Core.
resource "aws_kms_key" "license_signing" {
  description              = "Signs Synnax license keys (ML-DSA-44)"
  key_usage                = "SIGN_VERIFY"
  customer_master_key_spec = "ML_DSA_44"
  deletion_window_in_days  = 30
  enable_key_rotation      = false
}

resource "aws_kms_alias" "license_signing" {
  name          = "alias/synnax-license-signing"
  target_key_id = aws_kms_key.license_signing.key_id
}

# The identity the portal signs with. It can sign and read the public key, nothing else.
resource "aws_iam_user" "portal_signer" {
  name = "portal-signer"
}

data "aws_iam_policy_document" "portal_signer" {
  statement {
    actions   = ["kms:Sign", "kms:GetPublicKey"]
    resources = [aws_kms_key.license_signing.arn]
  }
}

resource "aws_iam_user_policy" "portal_signer" {
  name   = "sign-licenses"
  user   = aws_iam_user.portal_signer.name
  policy = data.aws_iam_policy_document.portal_signer.json
}

resource "aws_iam_access_key" "portal_signer" {
  user = aws_iam_user.portal_signer.name
}

# The Neon and Clerk Marketplace installs inject DATABASE_URL and the Clerk keys into
# the project. This root owns the rest of its environment.
resource "vercel_project" "portal" {
  name           = var.vercel_project_name
  framework      = "astro"
  root_directory = "site/portal"
  git_repository = {
    type              = "github"
    repo              = "synnaxlabs/synnax"
    production_branch = "main"
  }
}

resource "random_password" "cron_secret" {
  length  = 32
  special = false
}

locals {
  runtime_env = {
    AWS_REGION                   = var.aws_region
    AWS_ACCESS_KEY_ID            = aws_iam_access_key.portal_signer.id
    AWS_SECRET_ACCESS_KEY        = aws_iam_access_key.portal_signer.secret
    LICENSE_KMS_KEY_ARN          = aws_kms_key.license_signing.arn
    LICENSE_KID                  = "2"
    STAFF_ORG_ID                 = var.staff_org_id
    CLERK_WEBHOOK_SIGNING_SECRET = var.clerk_webhook_signing_secret
    RESEND_API_KEY               = var.resend_api_key
    MAIL_FROM                    = var.mail_from
    CRON_SECRET                  = random_password.cron_secret.result
  }
  sensitive_env = toset([
    "AWS_SECRET_ACCESS_KEY",
    "CLERK_WEBHOOK_SIGNING_SECRET",
    "RESEND_API_KEY",
    "CRON_SECRET",
  ])
}

resource "vercel_project_environment_variables" "portal" {
  project_id = vercel_project.portal.id
  variables = [
    for name, value in local.runtime_env : {
      key       = name
      value     = value
      target    = ["production", "preview"]
      sensitive = contains(local.sensitive_env, name)
    }
  ]
}

resource "vercel_project_domain" "portal" {
  project_id = vercel_project.portal.id
  domain     = var.domain
}

moved {
  from = aws_iam_user.hub_signer
  to   = aws_iam_user.portal_signer
}

moved {
  from = aws_iam_user_policy.hub_signer
  to   = aws_iam_user_policy.portal_signer
}

moved {
  from = aws_iam_access_key.hub_signer
  to   = aws_iam_access_key.portal_signer
}

moved {
  from = vercel_project_environment_variables.hub
  to   = vercel_project_environment_variables.portal
}

# The docs project owns docs.synnaxlabs.com. Terraform forgets the binding without
# removing it.
removed {
  from = vercel_project_domain.hub
  lifecycle {
    destroy = false
  }
}

# The floating CI license. Staff download it from the portal and hand it to Terraform
# as a variable; the workflows read the secret as SYNNAX_LICENSE_TOKEN.
resource "github_actions_secret" "license_token" {
  repository      = "synnax"
  secret_name     = "SYNNAX_LICENSE_TOKEN"
  plaintext_value = var.ci_license_token
}

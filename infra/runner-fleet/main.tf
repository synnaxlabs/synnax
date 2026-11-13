# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.


# IAM for the CI runner fleet. The roles were made by hand and are adopted with import
# blocks, so a hand edit now shows up as drift in `terraform plan`.

terraform {
  required_version = ">= 1.10"
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
  backend "s3" {
    bucket       = "synnax-agents-tfstate"
    key          = "runner-fleet/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}

# No default_tags: the live roles carry no tags, and new tags would show as changes.
provider "aws" {
  region = var.region
}

data "aws_caller_identity" "current" {}

locals {
  account = data.aws_caller_identity.current.account_id
  ec2_trust = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
  # Limits a statement to sessions of the sandbox instance.
  sandbox_session = "${aws_iam_role.sandbox.unique_id}:${var.sandbox_instance}"
  runner_os       = ["ubuntu", "windows"]
}

# The sandbox instance starts, stops, and inspects the runners.

import {
  to = aws_iam_role.sandbox
  id = "Runner-EC2-SSM-Role"
}

resource "aws_iam_role" "sandbox" {
  name = "Runner-EC2-SSM-Role"
  description = join(" ", [
    "Allows EC2 instances to call AWS services on your behalf",
    "(for runners).",
  ])
  assume_role_policy = local.ec2_trust
}

import {
  to = aws_iam_instance_profile.sandbox
  id = "Runner-EC2-SSM-Role"
}

resource "aws_iam_instance_profile" "sandbox" {
  name = "Runner-EC2-SSM-Role"
  role = aws_iam_role.sandbox.name
}

import {
  to = aws_iam_role_policy_attachment.sandbox_ssm
  id = "Runner-EC2-SSM-Role/arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "sandbox_ssm" {
  role       = aws_iam_role.sandbox.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

import {
  to = aws_iam_role_policy.describe_instances
  id = "Runner-EC2-SSM-Role:EC2DescribeInstances"
}

resource "aws_iam_role_policy" "describe_instances" {
  name = "EC2DescribeInstances"
  role = aws_iam_role.sandbox.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ec2:DescribeInstances", "ec2:DescribeVolumes"]
      Resource = "*"
    }]
  })
}

import {
  to = aws_iam_role_policy.runner_secrets_read
  id = "Runner-EC2-SSM-Role:RunnerSecretsRead"
}

resource "aws_iam_role_policy" "runner_secrets_read" {
  name = "RunnerSecretsRead"
  role = aws_iam_role.sandbox.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "ssm:GetParameter"
      Resource = "arn:aws:ssm:${var.region}:${local.account}:parameter/runners/*"
    }]
  })
}

import {
  to = aws_iam_role_policy.cloudwatch_read
  id = "Runner-EC2-SSM-Role:SandboxCloudWatchRead"
}

resource "aws_iam_role_policy" "cloudwatch_read" {
  name = "SandboxCloudWatchRead"
  role = aws_iam_role.sandbox.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "SandboxCloudWatchRead"
      Effect = "Allow"
      Action = [
        "cloudwatch:GetMetricData",
        "cloudwatch:ListMetrics",
        "logs:DescribeLogGroups",
        "logs:DescribeLogStreams",
        "logs:FilterLogEvents",
        "logs:GetLogEvents",
        "logs:StartLiveTail",
      ]
      Resource  = "*"
      Condition = { StringEquals = { "aws:userid" = local.sandbox_session } }
    }]
  })
}

import {
  to = aws_iam_role_policy.cost_explorer_read
  id = "Runner-EC2-SSM-Role:SandboxCostExplorerRead"
}

resource "aws_iam_role_policy" "cost_explorer_read" {
  name = "SandboxCostExplorerRead"
  role = aws_iam_role.sandbox.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = ["ce:GetCostAndUsage", "ce:GetCostForecast", "pricing:GetProducts"]
      Resource = "*"
    }]
  })
}

import {
  to = aws_iam_role_policy.runner_power
  id = "Runner-EC2-SSM-Role:SandboxRunnerPower"
}

resource "aws_iam_role_policy" "runner_power" {
  name = "SandboxRunnerPower"
  role = aws_iam_role.sandbox.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid      = "SandboxRunnerPower"
        Effect   = "Allow"
        Action   = ["ec2:StartInstances", "ec2:StopInstances"]
        Resource = "arn:aws:ec2:*:${local.account}:instance/*"
        Condition = {
          StringEquals = {
            "aws:userid"               = local.sandbox_session
            "ec2:ResourceTag/RunnerOS" = local.runner_os
          }
        }
      },
      # An agent seat stops itself, so the controller only starts it.
      {
        Sid      = "SandboxSeatStart"
        Effect   = "Allow"
        Action   = "ec2:StartInstances"
        Resource = "arn:aws:ec2:*:${local.account}:instance/*"
        Condition = {
          StringEquals = {
            "aws:userid"              = local.sandbox_session
            "ec2:ResourceTag/project" = "synnax-agents"
            "ec2:ResourceTag/pool"    = "agent"
          }
        }
      },
    ]
  })
}

import {
  to = aws_iam_role_policy.ssm_disk
  id = "Runner-EC2-SSM-Role:SandboxSSMDisk"
}

resource "aws_iam_role_policy" "ssm_disk" {
  name = "SandboxSSMDisk"
  role = aws_iam_role.sandbox.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect    = "Allow"
        Action    = ["ssm:DescribeInstanceInformation", "ssm:ListCommandInvocations"]
        Resource  = "*"
        Condition = { StringEquals = { "aws:userid" = local.sandbox_session } }
      },
      {
        Sid    = "RunDocuments"
        Effect = "Allow"
        Action = "ssm:SendCommand"
        Resource = [
          "arn:aws:ssm:*::document/AWS-RunPowerShellScript",
          "arn:aws:ssm:*::document/AWS-RunShellScript",
        ]
        Condition = { StringEquals = { "aws:userid" = local.sandbox_session } }
      },
      {
        Sid      = "RunOnRunnersOnly"
        Effect   = "Allow"
        Action   = "ssm:SendCommand"
        Resource = "arn:aws:ec2:*:${local.account}:instance/*"
        Condition = {
          StringEquals = {
            "aws:userid"               = local.sandbox_session
            "ssm:resourceTag/RunnerOS" = local.runner_os
          }
        }
      },
    ]
  })
}

# Every CI runner instance.

import {
  to = aws_iam_role.runner
  id = "EC2-SSM-Role"
}

resource "aws_iam_role" "runner" {
  name               = "EC2-SSM-Role"
  assume_role_policy = local.ec2_trust
}

import {
  to = aws_iam_instance_profile.runner
  id = "EC2-SSM-Profile"
}

resource "aws_iam_instance_profile" "runner" {
  name = "EC2-SSM-Profile"
  role = aws_iam_role.runner.name
}

import {
  to = aws_iam_role_policy_attachment.runner_cloudwatch
  id = "EC2-SSM-Role/arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

resource "aws_iam_role_policy_attachment" "runner_cloudwatch" {
  role       = aws_iam_role.runner.name
  policy_arn = "arn:aws:iam::aws:policy/CloudWatchAgentServerPolicy"
}

import {
  to = aws_iam_role_policy_attachment.runner_ssm
  id = "EC2-SSM-Role/arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "runner_ssm" {
  role       = aws_iam_role.runner.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

import {
  to = aws_iam_role_policy.deny_runner_secrets
  id = "EC2-SSM-Role:DenyRunnerSecrets"
}

# Runners run pull request content, so they must not read the secrets under /runners.
resource "aws_iam_role_policy" "deny_runner_secrets" {
  name = "DenyRunnerSecrets"
  role = aws_iam_role.runner.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "DenyRunnerSecrets"
      Effect = "Deny"
      Action = [
        "ssm:GetParameter",
        "ssm:GetParameters",
        "ssm:GetParametersByPath",
        "ssm:GetParameterHistory",
      ]
      Resource = [
        "arn:aws:ssm:*:${local.account}:parameter/runners",
        "arn:aws:ssm:*:${local.account}:parameter/runners/*",
      ]
    }]
  })
}

# Inline policies and managed attachments not declared here show up as drift, and an
# apply removes them.

resource "aws_iam_role_policies_exclusive" "sandbox" {
  role_name = aws_iam_role.sandbox.name
  policy_names = [
    aws_iam_role_policy.describe_instances.name,
    aws_iam_role_policy.runner_secrets_read.name,
    aws_iam_role_policy.cloudwatch_read.name,
    aws_iam_role_policy.cost_explorer_read.name,
    aws_iam_role_policy.runner_power.name,
    aws_iam_role_policy.ssm_disk.name,
  ]
}

resource "aws_iam_role_policy_attachments_exclusive" "sandbox" {
  role_name   = aws_iam_role.sandbox.name
  policy_arns = [aws_iam_role_policy_attachment.sandbox_ssm.policy_arn]
}

resource "aws_iam_role_policies_exclusive" "runner" {
  role_name    = aws_iam_role.runner.name
  policy_names = [aws_iam_role_policy.deny_runner_secrets.name]
}

resource "aws_iam_role_policy_attachments_exclusive" "runner" {
  role_name = aws_iam_role.runner.name
  policy_arns = [
    aws_iam_role_policy_attachment.runner_cloudwatch.policy_arn,
    aws_iam_role_policy_attachment.runner_ssm.policy_arn,
  ]
}

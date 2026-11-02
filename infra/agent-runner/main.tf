# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# One EC2 instance per agent CLI login (a seat), each with several GitHub Actions
# runners that share the login. A job starts a stopped instance through GitHub OIDC,
# and an idle instance stops itself, so its disk keeps the build caches warm.

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
    key          = "agent-runner/terraform.tfstate"
    region       = "us-east-1"
    use_lockfile = true
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = {
      project = "synnax-agents"
      owner   = "runners"
    }
  }
}

data "aws_ssm_parameter" "ubuntu" {
  name = join("/", [
    "/aws/service/canonical/ubuntu/server/24.04/stable/current",
    "amd64/hvm/ebs-gp3/ami-id",
  ])
}

data "aws_vpc" "default" {
  default = true
}

data "aws_subnets" "default" {
  filter {
    name   = "vpc-id"
    values = [data.aws_vpc.default.id]
  }
  filter {
    name   = "default-for-az"
    values = ["true"]
  }
  filter {
    name   = "availability-zone"
    values = var.zones
  }
}

locals {
  account = data.aws_caller_identity.current.account_id
  subnets = data.aws_subnets.default.ids
}

resource "aws_security_group" "runner" {
  name        = "synnax-agent-runner"
  description = "Agent runners: egress only, reached through SSM"
  vpc_id      = data.aws_vpc.default.id
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_iam_role" "runner" {
  name = "synnax-agent-runner"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Service = "ec2.amazonaws.com" }
      Action    = "sts:AssumeRole"
    }]
  })
}

resource "aws_iam_role_policy_attachment" "ssm" {
  role       = aws_iam_role.runner.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

data "aws_caller_identity" "current" {}

# Agents run pull request content, so they must not read the fleet controller's
# secrets under /runners.
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

resource "aws_iam_instance_profile" "runner" {
  name = "synnax-agent-runner"
  role = aws_iam_role.runner.name
}

resource "aws_instance" "seat" {
  for_each = var.seats
  ami      = data.aws_ssm_parameter.ubuntu.value
  # A hash of the name keeps each seat in its subnet when seats are added or removed.
  subnet_id = local.subnets[
    parseint(substr(md5(each.key), 0, 6), 16) % length(local.subnets)
  ]
  instance_type                        = var.instance_type
  vpc_security_group_ids               = [aws_security_group.runner.id]
  iam_instance_profile                 = aws_iam_instance_profile.runner.name
  instance_initiated_shutdown_behavior = "stop"
  user_data = templatefile("${path.module}/provision.sh", {
    idle_minutes = var.idle_minutes
    seat         = each.key
    agent        = each.value.agent
    runners      = each.value.runners
  })
  root_block_device {
    volume_type = "gp3"
    volume_size = var.disk_gb
    encrypted   = true
  }
  metadata_options {
    http_tokens = "required"
  }
  tags = {
    Name  = "synnax-agent-${each.key}"
    pool  = "agent"
    agent = each.value.agent
    seat  = each.key
  }
  # A new AMI or bootstrap script must not wipe the warm caches or the login. Replace
  # an instance on purpose with `terraform apply -replace`.
  lifecycle {
    ignore_changes = [ami, user_data]
  }
}

data "aws_iam_openid_connect_provider" "github" {
  url = "https://token.actions.githubusercontent.com"
}

# Lets workflows in the repository start a stopped runner, and nothing else.
resource "aws_iam_role" "start" {
  name = "synnax-agent-runner-start"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = data.aws_iam_openid_connect_provider.github.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
        }
        StringLike = {
          "token.actions.githubusercontent.com:sub" = "repo:${var.repository}:*"
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "start" {
  name = "start-agent-runners"
  role = aws_iam_role.start.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "ec2:DescribeInstances"
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = "ec2:StartInstances"
        Resource = "arn:aws:ec2:${var.region}:*:instance/*"
        Condition = {
          StringEquals = {
            "aws:ResourceTag/project" = "synnax-agents"
            "aws:ResourceTag/pool"    = "agent"
          }
        }
      },
    ]
  })
}

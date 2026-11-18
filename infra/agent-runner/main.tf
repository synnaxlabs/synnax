# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# One EC2 instance per seat, each with several GitHub Actions runners. The fleet
# controller starts a stopped seat for a queued job, on a shared Claude login that it
# picks. An idle instance stops itself, so its disk keeps the build caches warm.

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
  # A seat writes limited-until at a usage limit and login-broken at a rejected token,
  # Check - Runners writes the sign-in state, and the fleet controller writes the login,
  # so an apply must keep these tags.
  ignore_tags {
    keys = ["limited-until", "login-broken", "auth", "auth-checked", "login"]
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
  # The shared Claude logins, stored by `seat.sh token`. They stay in us-east-1 for
  # seats in any region.
  logins = join(":", [
    "arn:aws:secretsmanager:us-east-1", local.account, "secret:synnax-agents/logins/*",
  ])
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

# Lets a seat record when the usage limit of its login resets, and when Claude rejected
# its token. The fleet controller moves each to the login, which no seat then starts on.
# A seat can tag only its own instance: the user ID of an instance role session must end
# with the ID of the tagged instance.
resource "aws_iam_role_policy" "limited_until" {
  name = "TagLimitedUntil"
  role = aws_iam_role.runner.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "ec2:CreateTags"
      Resource = "arn:aws:ec2:${var.region}:${local.account}:instance/*"
      Condition = {
        StringEquals = {
          "aws:ResourceTag/project" = "synnax-agents"
          "aws:ResourceTag/pool"    = ["agent", "review"]
          "aws:userid" = join(":", [
            aws_iam_role.runner.unique_id, "$${ec2:InstanceID}",
          ])
        }
        "ForAllValues:StringEquals" = {
          "aws:TagKeys" = ["limited-until", "login-broken"]
        }
      }
    }]
  })
}

# Lets a seat read the token of the Claude login that the fleet controller gave it.
# The controller adds the seat's session to the login's `holders` tag, and removes it
# once the seat stops. A tag value fits 6 sessions, so the controller gives a login to
# at most 6 seats.
resource "aws_iam_role_policy" "login" {
  name = "ReadHeldLogin"
  role = aws_iam_role.runner.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "secretsmanager:GetSecretValue"
      Resource = local.logins
      Condition = {
        StringLike = { "secretsmanager:ResourceTag/holders" = "*$${aws:userid}*" }
      }
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
  instance_type                        = var.pools[each.value.pool].instance_type
  vpc_security_group_ids               = [aws_security_group.runner.id]
  iam_instance_profile                 = aws_iam_instance_profile.runner.name
  instance_initiated_shutdown_behavior = "stop"
  user_data = templatefile("${path.module}/provision.sh", {
    idle_minutes = var.pools[each.value.pool].idle_minutes
    pool         = each.value.pool
  })
  root_block_device {
    volume_type = "gp3"
    volume_size = var.pools[each.value.pool].disk_gb
    encrypted   = true
  }
  # Jobs read the instance's `login` tag from the metadata service.
  metadata_options {
    http_tokens            = "required"
    instance_metadata_tags = "enabled"
  }
  tags = {
    Name    = "synnax-agent-${each.key}"
    pool    = each.value.pool
    agent   = each.value.agent
    seat    = each.key
    runners = var.pools[each.value.pool].runners
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

# Lets workflows in the repository see the runners, and nothing else.
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
    ]
  })
}

# Lets Agent - Requeue read the tags of the shared Claude logins, never their tokens.
resource "aws_iam_role_policy" "start_logins" {
  name = "ReadLoginTags"
  role = aws_iam_role.start.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "secretsmanager:ListSecrets"
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = "secretsmanager:DescribeSecret"
        Resource = join(":", [
          "arn:aws:secretsmanager:us-east-1",
          local.account,
          "secret:synnax-agents/logins/*",
        ])
      },
    ]
  })
}

# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# One EC2 host that runs a Core container per pull request behind Caddy. CI pushes
# images to ECR and deploys through SSM, with a role it assumes through GitHub OIDC.

terraform {
  required_version = ">= 1.10"
  # Created by the config in state/.
  backend "s3" {
    bucket = "synnax-preview-terraform-004704223095"
    key    = "preview/terraform.tfstate"
    region = "us-east-1"
    # The bucket encrypts with its KMS key. encrypt = true would ask for AES256.
    encrypt      = false
    use_lockfile = true
  }
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 6.0"
    }
  }
}

provider "aws" {
  region = var.region
  default_tags {
    tags = { project = "synnax-agents", owner = "preview" }
  }
}

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "domain" {
  description = <<-EOT
    Parent domain of the previews. Terraform creates its Route 53 zone, which the
    parent zone must delegate to. A preview is at pr-<number>.<domain>.
  EOT
  type        = string
  default     = "preview.synnaxlabs.com"
}

variable "repository" {
  description = "GitHub repository that may deploy previews."
  type        = string
  default     = "synnaxlabs/synnax"
}

variable "oidc_provider_arn" {
  description = "ARN of an existing GitHub Actions OIDC provider. Empty creates one."
  type        = string
  default     = ""
}

variable "license_parameter" {
  description = "SSM SecureString parameter that holds the Core license key."
  type        = string
  default     = "/synnax-preview/license-key"
}

variable "instance_type" {
  type    = string
  default = "t3.large"
}

variable "ttl_days" {
  description = "Days with no deploy or view before a preview marked none is removed."
  type        = number
  default     = 2
}

# The defaults fit a t3.large (2 vCPUs, 8 GiB). Twenty-five previews may take 6400 MiB,
# which leaves about 1.4 GiB for Caddy, Docker, and the SSM agent. An idle Core uses
# about 65 MiB and 1% of a vCPU, so the CPU limit is a ceiling for a starting Core. The
# sample simulator stores about 16 MiB a day in memory. GOMEMLIMIT makes the Go GC hold
# the heap under the container limit, so a Core keeps about 8 days of it.
variable "max_previews" {
  description = "Previews the host runs at once. At the cap, a new one evicts one."
  type        = number
  default     = 25
}

variable "preview_cpus" {
  description = "CPUs each preview Core may use, as docker run --cpus takes them."
  type        = string
  default     = "0.25"
}

variable "preview_memory" {
  description = "Memory each preview Core may use, as docker run --memory takes it."
  type        = string
  default     = "256m"
}

variable "preview_go_memory" {
  description = "GOMEMLIMIT of each preview Core. Keep it below preview_memory."
  type        = string
  default     = "200MiB"
}

data "aws_ssm_parameter" "ami" {
  name = "/aws/service/ami-amazon-linux-latest/al2023-ami-kernel-default-x86_64"
}

resource "aws_ecr_repository" "preview" {
  name                 = "synnax-preview"
  image_tag_mutability = "IMMUTABLE"
  force_delete         = true
  image_scanning_configuration {
    scan_on_push = true
  }
  encryption_configuration {
    encryption_type = "KMS"
  }
}

resource "aws_ecr_lifecycle_policy" "preview" {
  repository = aws_ecr_repository.preview.name
  policy = jsonencode({
    rules = [{
      rulePriority = 1
      description  = "Expire images older than the preview TTL"
      selection = {
        tagStatus   = "any"
        countType   = "sinceImagePushed"
        countUnit   = "days"
        countNumber = var.ttl_days
      }
      action = { type = "expire" }
    }]
  })
}

resource "aws_security_group" "preview" {
  name        = "synnax-preview"
  description = "HTTPS to Caddy"
  ingress {
    description = "HTTPS to Caddy"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
  # AWS APIs, image registries, packages, and ACME all use HTTPS. The VPC resolver and
  # Amazon Time Sync need no rule.
  egress {
    description = "HTTPS to AWS, registries, and ACME"
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

data "aws_iam_policy_document" "host_trust" {
  statement {
    actions = ["sts:AssumeRole"]
    principals {
      type        = "Service"
      identifiers = ["ec2.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "host" {
  name               = "synnax-preview-host"
  assume_role_policy = data.aws_iam_policy_document.host_trust.json
}

resource "aws_iam_role_policy_attachment" "host_ssm" {
  role       = aws_iam_role.host.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonSSMManagedInstanceCore"
}

resource "aws_iam_role_policy_attachment" "host_ecr" {
  role       = aws_iam_role.host.name
  policy_arn = "arn:aws:iam::aws:policy/AmazonEC2ContainerRegistryReadOnly"
}

data "aws_caller_identity" "current" {}

resource "aws_iam_role_policy" "host_license" {
  role = aws_iam_role.host.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Action = "ssm:GetParameter"
      Resource = join("", [
        "arn:aws:ssm:${var.region}:${data.aws_caller_identity.current.account_id}",
        ":parameter/${trimprefix(var.license_parameter, "/")}",
      ])
    }]
  })
}

# lego on the host gets the wildcard certificate through a DNS-01 challenge.
resource "aws_iam_role_policy" "host_dns" {
  role = aws_iam_role.host.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "route53:ChangeResourceRecordSets",
          "route53:ListResourceRecordSets",
        ]
        Resource = aws_route53_zone.preview.arn
      },
      {
        Effect   = "Allow"
        Action   = "route53:GetChange"
        Resource = "arn:aws:route53:::change/*"
      },
    ]
  })
}

resource "aws_iam_instance_profile" "host" {
  name = "synnax-preview-host"
  role = aws_iam_role.host.name
}

resource "aws_instance" "host" {
  ami                    = data.aws_ssm_parameter.ami.value
  instance_type          = var.instance_type
  ebs_optimized          = true
  monitoring             = true
  iam_instance_profile   = aws_iam_instance_profile.host.name
  vpc_security_group_ids = [aws_security_group.preview.id]
  user_data = templatefile("${path.module}/bootstrap.sh", {
    env       = local.host_env
    caddyfile = file("${path.module}/Caddyfile")
    preview   = file("${path.module}/preview.sh")
  })
  root_block_device {
    volume_size = 60
    volume_type = "gp3"
  }
  metadata_options {
    http_tokens = "required"
  }
  tags = { Name = "synnax-preview" }
  # User data only bootstraps a new host. install.sh brings a running host up to date,
  # so a script or setting change never replaces the host under live previews.
  lifecycle {
    ignore_changes = [ami, user_data]
  }
}

locals {
  host_env = <<-EOT
    PREVIEW_DOMAIN=${var.domain}
    PREVIEW_ZONE_ID=${aws_route53_zone.preview.zone_id}
    PREVIEW_ACME=/etc/caddy/acme
    PREVIEW_SITES=/etc/caddy/sites
    PREVIEW_TTL_DAYS=${var.ttl_days}
    PREVIEW_LICENSE_PARAMETER=${var.license_parameter}
    PREVIEW_REGION=${var.region}
    PREVIEW_MAX=${var.max_previews}
    PREVIEW_CPUS=${var.preview_cpus}
    PREVIEW_MEMORY=${var.preview_memory}
    PREVIEW_GO_MEMORY=${var.preview_go_memory}
    PREVIEW_LOGS=/var/log/caddy
  EOT
}

resource "aws_eip" "host" {
  domain = "vpc"
}

resource "aws_eip_association" "host" {
  allocation_id = aws_eip.host.id
  instance_id   = aws_instance.host.id
}

resource "aws_route53_zone" "preview" {
  # checkov:skip=CKV2_AWS_38:The parent zone is unsigned, so signing adds no trust.
  # checkov:skip=CKV2_AWS_39:The zone holds one record to a public host.
  name = var.domain
}

resource "aws_route53_record" "previews" {
  zone_id = aws_route53_zone.preview.zone_id
  name    = "*.${var.domain}"
  type    = "A"
  ttl     = 300
  records = [aws_eip.host.public_ip]
}

resource "aws_iam_openid_connect_provider" "github" {
  count          = var.oidc_provider_arn == "" ? 1 : 0
  url            = "https://token.actions.githubusercontent.com"
  client_id_list = ["sts.amazonaws.com"]
}

locals {
  oidc_provider_arn = coalesce(
    var.oidc_provider_arn,
    one(aws_iam_openid_connect_provider.github[*].arn),
  )
}

# GitHub gives an OIDC token to a pull request workflow only for a branch in this
# repository, never for a fork, and the subject names the pull_request event.
data "aws_iam_policy_document" "ci_trust" {
  statement {
    actions = ["sts:AssumeRoleWithWebIdentity"]
    principals {
      type        = "Federated"
      identifiers = [local.oidc_provider_arn]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:aud"
      values   = ["sts.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "token.actions.githubusercontent.com:sub"
      values   = ["repo:${var.repository}:pull_request"]
    }
  }
}

locals {
  image = join("", [
    replace(aws_ecr_repository.preview.repository_url, ".", "\\."),
    ":pr-[0-9]+-[0-9a-f]{40}-[0-9]+-[0-9]+",
  ])
  notice_time = "[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}Z"
}

# Any workflow on a pull request branch can assume the CI role, so the role may send
# only this document. Its pattern admits the preview commands that preview.yaml sends,
# and no shell syntax.
resource "aws_ssm_document" "preview" {
  name            = "synnax-preview"
  document_type   = "Command"
  document_format = "JSON"
  content = jsonencode({
    schemaVersion = "2.2"
    description   = "Runs one preview command on the preview host."
    parameters = {
      commands = {
        type        = "String"
        description = "The preview command line."
        allowedPattern = join("", [
          "^/usr/local/bin/preview (",
          "up [0-9]+ ${local.image} (human|none) [0-9]+",
          "|mark [0-9]+ (human|none) [0-9]+",
          "|down [0-9]+",
          "|notices",
          "|ack( [0-9]+ ${local.notice_time})+",
          ")$",
        ])
      }
    }
    mainSteps = [{
      action = "aws:runShellScript"
      name   = "preview"
      inputs = { runCommand = ["{{ commands }}"] }
    }]
  })
}

data "aws_iam_policy_document" "ci" {
  statement {
    actions   = ["ecr:GetAuthorizationToken"]
    resources = ["*"]
  }
  statement {
    actions = [
      "ecr:BatchCheckLayerAvailability",
      "ecr:BatchGetImage",
      "ecr:CompleteLayerUpload",
      "ecr:InitiateLayerUpload",
      "ecr:PutImage",
      "ecr:UploadLayerPart",
    ]
    resources = [aws_ecr_repository.preview.arn]
  }
  statement {
    actions   = ["ssm:SendCommand"]
    resources = [aws_instance.host.arn, aws_ssm_document.preview.arn]
  }
  statement {
    actions   = ["ssm:GetCommandInvocation"]
    resources = ["*"]
  }
}

resource "aws_iam_role" "ci" {
  name                 = "synnax-preview-ci"
  assume_role_policy   = data.aws_iam_policy_document.ci_trust.json
  max_session_duration = 3600
}

resource "aws_iam_role_policy" "ci" {
  role   = aws_iam_role.ci.id
  policy = data.aws_iam_policy_document.ci.json
}

output "ci_role_arn" {
  value = aws_iam_role.ci.arn
}

output "instance_id" {
  value = aws_instance.host.id
}

output "public_ip" {
  value = aws_eip.host.public_ip
}

output "domain" {
  value = var.domain
}

output "name_servers" {
  description = "The NS records the parent zone needs for <domain>."
  value       = aws_route53_zone.preview.name_servers
}

output "ecr_repository_url" {
  value = aws_ecr_repository.preview.repository_url
}

output "host_env" {
  description = "The settings install.sh writes to the host."
  value       = local.host_env
}

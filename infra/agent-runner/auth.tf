# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

# Lets Check - Runners read whether each Codex seat's CLI is signed in. Claude seats run
# on a shared login. The role runs only this document, which prints one word and never
# the login details.

resource "aws_ssm_document" "auth" {
  name            = "synnax-agent-seat-auth"
  document_type   = "Command"
  document_format = "YAML"
  content = yamlencode({
    schemaVersion = "2.2"
    description   = "Prints signed-in or signed-out for the seat's Codex CLI"
    parameters = {
      agent = { type = "String", allowedValues = ["codex"] }
    }
    mainSteps = [{
      action = "aws:runShellScript"
      name   = "status"
      inputs = {
        timeoutSeconds = "60"
        runCommand = [<<-EOF
          case $(sudo -iu runner codex login status 2>&1) in
            "Logged in"*) echo signed-in ;;
            "Not logged in"*) echo signed-out ;;
          esac
        EOF
        ]
      }
    }]
  })
}

resource "aws_iam_role" "auth" {
  name = "synnax-agent-seat-auth"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Principal = { Federated = data.aws_iam_openid_connect_provider.github.arn }
      Action    = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        # Only workflow runs on main, such as the schedule, may assume it.
        StringEquals = {
          "token.actions.githubusercontent.com:aud" = "sts.amazonaws.com"
          "token.actions.githubusercontent.com:sub" = join(":", [
            "repo", var.repository, "ref", "refs/heads/main",
          ])
        }
      }
    }]
  })
}

resource "aws_iam_role_policy" "auth" {
  name = "check-seat-auth"
  role = aws_iam_role.auth.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "ec2:DescribeInstances",
          "ssm:DescribeInstanceInformation",
          "ssm:ListCommandInvocations",
        ]
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = "ssm:SendCommand"
        Resource = aws_ssm_document.auth.arn
      },
      {
        Effect    = "Allow"
        Action    = "ssm:SendCommand"
        Resource  = "arn:aws:ec2:${var.region}:${local.account}:instance/*"
        Condition = { StringEquals = { "ssm:resourceTag/project" = "synnax-agents" } }
      },
      {
        Effect   = "Allow"
        Action   = "ec2:CreateTags"
        Resource = "arn:aws:ec2:${var.region}:${local.account}:instance/*"
        Condition = {
          StringEquals                = { "aws:ResourceTag/project" = "synnax-agents" }
          "ForAllValues:StringEquals" = { "aws:TagKeys" = ["auth", "auth-checked"] }
        }
      },
    ]
  })
}

# Lets Check - Runners read the tags of the shared Claude logins, never their tokens.
resource "aws_iam_role_policy" "auth_logins" {
  name = "check-login-tags"
  role = aws_iam_role.auth.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect   = "Allow"
        Action   = "secretsmanager:ListSecrets"
        Resource = "*"
      },
      {
        Effect   = "Allow"
        Action   = "secretsmanager:DescribeSecret"
        Resource = local.logins
      },
    ]
  })
}

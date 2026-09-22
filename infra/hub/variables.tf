# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

variable "aws_region" {
  type    = string
  default = "us-east-1"
}

variable "vercel_team_id" {
  description = "The Vercel team that owns the hub project."
  type        = string
}

variable "vercel_project_name" {
  description = "The Vercel project the hub deploys to."
  type        = string
  default     = "site"
}

variable "domain" {
  type    = string
  default = "docs.synnaxlabs.com"
}

variable "staff_org_id" {
  description = "Clerk id of the Synnax Labs team organization. Its admins are staff."
  type        = string
}

variable "clerk_webhook_signing_secret" {
  description = "Signing secret of the Clerk webhook that points at /api/webhooks/clerk."
  type        = string
  sensitive   = true
}

variable "resend_api_key" {
  type      = string
  sensitive = true
}

variable "mail_from" {
  type    = string
  default = "Synnax Labs <licenses@synnaxlabs.com>"
}

variable "ci_license_token" {
  description = "A floating token issued to the Synnax Labs organization for CI."
  type        = string
  sensitive   = true
}

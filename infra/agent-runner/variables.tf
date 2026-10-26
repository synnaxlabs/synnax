# Copyright 2026 Synnax Labs, Inc.
#
# Use of this software is governed by the Business Source License included in the file
# licenses/BSL.txt.
#
# As of the Change Date specified in that file, in accordance with the Business Source
# License, use of this software will be governed by the Apache License, Version 2.0,
# included in the file licenses/APL.txt.

variable "region" {
  type    = string
  default = "us-east-1"
}

variable "repository" {
  description = "GitHub repository whose workflows may start a runner"
  type        = string
  default     = "synnaxlabs/synnax"
}

variable "instances" {
  description = "Size of the pool: the most agent jobs that run at once"
  type        = number
  default     = 2
}

variable "zones" {
  description = "Zones with capacity for instance_type: a stopped runner restarts there"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d"]
}

variable "instance_type" {
  type    = string
  default = "c7i.4xlarge"
}

variable "disk_gb" {
  type    = number
  default = 250
}

variable "idle_minutes" {
  description = "Minutes without a job before an instance stops itself"
  type        = number
  default     = 20
}

output "instance_ids" {
  value = aws_instance.runner[*].id
}

output "start_role_arn" {
  description = "Value for the AGENT_RUNNER_ROLE_ARN repository variable"
  value       = aws_iam_role.start.arn
}

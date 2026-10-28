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

# A seat is one Claude Max or ChatGPT login. Each runner on a seat runs one job at a
# time, and a Claude seat adds one runner reserved for Label - Classify.
variable "seats" {
  description = "Agent CLI logins by name, one instance each"
  type = map(object({
    agent   = string
    runners = optional(number, 3)
  }))
  default = {
    claude-1 = { agent = "claude" }
  }
  validation {
    condition = alltrue([
      for name, s in var.seats :
      can(regex("^[a-z0-9-]+$", name)) && contains(["claude", "codex"], s.agent) &&
      s.runners > 0 && floor(s.runners) == s.runners
    ])
    error_message = join(" ", [
      "Seat names use a-z, 0-9, and dashes. Agents are claude or codex.",
      "Runner counts are positive whole numbers.",
    ])
  }
}

variable "zones" {
  description = "Zones with capacity for instance_type: a stopped runner restarts there"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d"]
}

variable "instance_type" {
  type = string
  # Three agents build at once, so memory runs out before CPU.
  default = "m7i.4xlarge"
}

variable "disk_gb" {
  type    = number
  default = 400
}

variable "idle_minutes" {
  description = "Minutes without a job before an instance stops itself"
  type        = number
  default     = 20
}

output "instance_ids" {
  value = { for name, i in aws_instance.seat : name => i.id }
}

output "start_role_arn" {
  description = "Value for the AGENT_RUNNER_ROLE_ARN repository variable"
  value       = aws_iam_role.start.arn
}

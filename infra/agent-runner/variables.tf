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
# time, and a Claude seat in the agent pool adds one runner reserved for Label -
# Classify. A review seat's runners carry only the `review` label, so agent jobs never
# land on them.
variable "seats" {
  description = "Agent CLI logins by name, one instance each"
  type = map(object({
    agent   = string
    runners = optional(number, 3)
    pool    = optional(string, "agent")
  }))
  default = {
    claude-1 = { agent = "claude" }
    review-1 = { agent = "claude", runners = 6, pool = "review" }
  }
  validation {
    condition = alltrue([
      for name, s in var.seats :
      can(regex("^[a-z0-9-]+$", name)) && contains(["claude", "codex"], s.agent) &&
      s.runners > 0 && floor(s.runners) == s.runners &&
      contains(keys(var.pools), s.pool)
    ])
    error_message = join(" ", [
      "Seat names use a-z, 0-9, and dashes. Agents are claude or codex.",
      "Runner counts are positive whole numbers. Pools are keys of var.pools.",
    ])
  }
}

variable "pools" {
  description = "Instance shape by pool, with the idle minutes before it stops itself"
  type = map(object({
    instance_type = string
    disk_gb       = number
    idle_minutes  = number
  }))
  default = {
    # Three agents build at once, so memory runs out before CPU.
    agent = { instance_type = "m7i.4xlarge", disk_gb = 400, idle_minutes = 20 }
    # Reviews only read code. The longer idle time spares most reviews in a workday
    # the wait for a stopped instance to boot.
    review = { instance_type = "m7i.xlarge", disk_gb = 100, idle_minutes = 60 }
  }
}

variable "zones" {
  description = "Zones with capacity for every pool: a stopped runner restarts there"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b", "us-east-1c", "us-east-1d"]
}

output "instance_ids" {
  value = { for name, i in aws_instance.seat : name => i.id }
}

output "start_role_arn" {
  description = "Value for the AGENT_RUNNER_ROLE_ARN repository variable"
  value       = aws_iam_role.start.arn
}

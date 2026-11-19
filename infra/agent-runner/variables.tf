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

# A seat is one machine. A Claude seat runs on a shared login that the fleet controller
# gives it when it starts, and a Codex seat holds its own ChatGPT login. A review seat's
# runners carry only the `review` label, so agent jobs never land on them.
variable "seats" {
  description = "Seats by name, one instance each"
  type = map(object({
    agent = string
    pool  = optional(string, "agent")
  }))
  default = {
    claude-1 = { agent = "claude" }
    review-1 = { agent = "claude", pool = "review" }
  }
  validation {
    condition = alltrue([
      for name, s in var.seats :
      can(regex("^[a-z0-9-]+$", name)) && contains(["claude", "codex"], s.agent) &&
      contains(keys(var.pools), s.pool)
    ])
    error_message = join(" ", [
      "Seat names use a-z, 0-9, and dashes. Agents are claude or codex.",
      "Pools are keys of var.pools.",
    ])
  }
}

# Each runner runs one job at a time, and every job on a seat shares its login. A
# change to runners reaches a seat at its next `seat.sh register`, which adds runners
# but never removes them.
variable "pools" {
  description = "Instance shape, runner count, and minutes idle before a stop, by pool"
  type = map(object({
    instance_type = string
    disk_gb       = number
    idle_minutes  = number
    runners       = number
  }))
  default = {
    # Five agents build at once at up to 11 GiB each, so memory runs out before CPU.
    agent = {
      instance_type = "r7i.4xlarge", disk_gb = 400, idle_minutes = 20, runners = 5
    }
    # Reviews only read code. The longer idle time spares most reviews in a workday
    # the wait for a stopped instance to boot.
    review = {
      instance_type = "m7i.xlarge", disk_gb = 100, idle_minutes = 60, runners = 12
    }
  }
  validation {
    condition = alltrue([
      for p in var.pools : p.runners > 0 && floor(p.runners) == p.runners
    ])
    error_message = "Runner counts are positive whole numbers."
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

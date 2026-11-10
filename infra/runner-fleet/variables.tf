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

variable "sandbox_instance" {
  description = "ID of the Sandbox-Windows instance that manages the runners"
  type        = string
  default     = "i-0daf42c1631de819a"
}

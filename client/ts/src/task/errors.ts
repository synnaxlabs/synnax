// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@/status";

/**
 * An error built from a status a task returned, such as the answer to a command. A
 * status made from it with {@link status.fromException} keeps the status message and
 * description.
 */
export class StatusError extends Error implements status.Custom {
  readonly description: string;

  constructor(stat: Pick<status.Status, "message" | "description">) {
    super(stat.message);
    this.description = stat.description;
  }

  toStatus(): Pick<status.Status, "message" | "description"> {
    return { message: this.message, description: this.description };
  }
}

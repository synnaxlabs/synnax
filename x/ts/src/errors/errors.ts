// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

class Canceled extends Error {
  static readonly MESSAGE = "canceled";
  constructor() {
    super(Canceled.MESSAGE);
  }

  /** Returns true if the error or message is a cancellation error" */
  matches(e: Error | string): boolean {
    if (typeof e === "string") return e.includes(Canceled.MESSAGE);
    return e instanceof Canceled || e.message.includes(Canceled.MESSAGE);
  }
}

/**
 * CANCELED should be thrown to indicate the cancellation of an operation.
 */
export const CANCELED = new Canceled();

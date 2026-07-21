// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type destructor, errors } from "@synnaxlabs/x";

/**
 * Accumulates rollbacks for optimistic cache writes. Write methods add each
 * store mutation's undo and run them all when the network call fails.
 */
export class Rollback {
  private readonly destructors: destructor.Destructor[] = [];

  /** Adds rollbacks, run in reverse order of addition. */
  add(...destructors: destructor.Destructor[]): void {
    this.destructors.push(...destructors);
  }

  /** Runs every rollback in reverse order. Errors are logged, not thrown. */
  private run(): void {
    try {
      this.destructors.reverse().forEach((d) => d());
    } catch (error) {
      console.error("failed to roll back optimistic writes", error);
    } finally {
      this.destructors.length = 0;
    }
  }

  /**
   * Runs the network call, rolling back accumulated writes and rethrowing on
   * failure.
   */
  async guard<T>(call: () => Promise<T>): Promise<T> {
    try {
      return await call();
    } catch (error) {
      this.run();
      throw errors.fromUnknown(error);
    }
  }
}

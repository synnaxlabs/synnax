// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@/errors";

export interface Destructor {
  (): void;
}

export interface Async {
  (): Promise<void>;
}

export const NOOP = () => {};

/**
 * Runs every destructor in reverse of the given order, so callers pass them in
 * acquisition order and later teardown sees the earlier resources still alive.
 * Nullish entries are skipped. A throw is logged and the rest still run, so one
 * failed teardown cannot strand the others.
 */
export const unwind = (...destructors: Array<Destructor | undefined | null>): void => {
  for (let i = destructors.length - 1; i >= 0; i--)
    try {
      destructors[i]?.();
    } catch (error) {
      console.error("destructor failed", error);
    }
};

/** Accumulates destructors and runs them all when a guarded call fails. */
export class Chain {
  private readonly destructors: Destructor[] = [];

  /** Adds destructors, run in reverse order of addition. */
  add(...destructors: Destructor[]): void {
    this.destructors.push(...destructors);
  }

  /**
   * Runs the call, unwinding accumulated destructors and rethrowing on
   * failure. Destructor errors are logged, not thrown.
   */
  async guard<T>(call: () => Promise<T>): Promise<T> {
    try {
      return await call();
    } catch (error) {
      unwind(...this.destructors);
      this.destructors.length = 0;
      throw errors.fromUnknown(error);
    }
  }
}

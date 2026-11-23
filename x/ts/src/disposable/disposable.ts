// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Runtimes without the symbols get the keys that Babel, esbuild, and Oxc fall back to
// when they transpile `using` declarations.
const DISPOSE =
  typeof Symbol.dispose === "symbol" ? Symbol.dispose : Symbol.for("Symbol.dispose");
const ASYNC_DISPOSE =
  typeof Symbol.asyncDispose === "symbol"
    ? Symbol.asyncDispose
    : Symbol.for("Symbol.asyncDispose");

interface Closer<R> {
  close: () => R;
}

/**
 * Gives the instances of a class with a synchronous `close` method a
 * `[Symbol.dispose]` and a `[Symbol.asyncDispose]` method, both of which call `close`.
 * Call it in a static block of the class, and declare the two methods on the class
 * for the type checker. It does not mutate the global `Symbol`.
 */
export const implement = (cls: { prototype: Closer<void> }): void => {
  Object.defineProperty(cls.prototype, DISPOSE, {
    value(this: Closer<void>): void {
      this.close();
    },
    writable: true,
    configurable: true,
  });
  Object.defineProperty(cls.prototype, ASYNC_DISPOSE, {
    async value(this: Closer<void>): Promise<void> {
      this.close();
    },
    writable: true,
    configurable: true,
  });
};

/**
 * Gives the instances of a class with an asynchronous `close` method a
 * `[Symbol.asyncDispose]` method that calls `close`. Call it in a static block of the
 * class, and declare the method on the class for the type checker. It does not
 * mutate the global `Symbol`.
 */
export const implementAsync = (cls: { prototype: Closer<Promise<void>> }): void => {
  Object.defineProperty(cls.prototype, ASYNC_DISPOSE, {
    async value(this: Closer<Promise<void>>): Promise<void> {
      await this.close();
    },
    writable: true,
    configurable: true,
  });
};

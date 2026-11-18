// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** @returns A promise that resolves on the next microtask. */
export const nextMicrotask = (): Promise<void> => Promise.resolve();

/** @returns A promise that resolves on the next task, after pending microtasks run. */
export const nextTask = (): Promise<void> =>
  new Promise((resolve) => setTimeout(resolve, 0));

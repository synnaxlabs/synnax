// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { onTestFinished } from "vitest";

/**
 * pinLocationOrigin pins window.location.origin for the duration of the calling
 * test. jsdom's location is not assignable, so code that branches on the page origin
 * cannot be driven any other way. The original location is restored automatically
 * when the test finishes.
 */
export const pinLocationOrigin = (origin: string): void => {
  const original = window.location;
  Object.defineProperty(window, "location", {
    configurable: true,
    value: { origin },
  });
  onTestFinished(() => {
    Object.defineProperty(window, "location", {
      configurable: true,
      value: original,
    });
  });
};

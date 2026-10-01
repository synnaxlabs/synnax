// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, onTestFinished, TestRunner } from "vitest";

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

const disable = (): void => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = false;
};

/**
 * Turns off React's act environment for the rest of the calling test, or for the rest
 * of the spec file when called outside a test. Call it for a tree wired to a live Core,
 * whose updates cannot be scoped to an act.
 */
export const disableActEnvironment = (): void => {
  if (TestRunner.getCurrentTest() != null) {
    const previous = globalThis.IS_REACT_ACT_ENVIRONMENT;
    disable();
    onTestFinished(() => {
      globalThis.IS_REACT_ACT_ENVIRONMENT = previous;
    });
    return;
  }
  // Testing Library turns the environment on in a beforeAll, which runs after a call
  // made while vitest collects the file.
  disable();
  beforeAll(disable);
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll, TestRunner } from "vitest";

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

const disable = (): void => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = false;
};

/**
 * Turns off React's act environment for the rest of the spec file. Call it for a tree
 * wired to a live Core, whose updates cannot be scoped to an act and can land after the
 * test that started them ends.
 */
export const disableActEnvironment = (): void => {
  disable();
  // Testing Library turns the environment on in a beforeAll, which runs after a call
  // made while vitest collects the file.
  if (TestRunner.getCurrentTest() == null) beforeAll(disable);
};

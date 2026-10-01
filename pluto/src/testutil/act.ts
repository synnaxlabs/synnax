// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { beforeAll } from "vitest";

declare global {
  var IS_REACT_ACT_ENVIRONMENT: boolean | undefined;
}

const disable = (): void => {
  globalThis.IS_REACT_ACT_ENVIRONMENT = false;
};

/**
 * Turns off React's act environment for the rest of the spec file, so React renders
 * updates as it does in production. Call it for a tree wired to a live Core, whose
 * updates arrive at any time and cannot be scoped to an act. Testing Library's renders
 * and events still run inside act.
 */
export const disableActEnvironment = (): void => {
  // Testing Library turns the environment on in a beforeAll, which runs after a call
  // made while vitest collects the file. A call made inside a test lands after it.
  disable();
  beforeAll(disable);
};

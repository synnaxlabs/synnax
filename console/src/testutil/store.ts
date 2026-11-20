// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { configureStore } from "@reduxjs/toolkit";

// The checks still run. Only their timing warnings, which track machine load, go.
const STORE_CHECKS = {
  immutableCheck: { warnAfter: Infinity },
  serializableCheck: { warnAfter: Infinity },
};

/**
 * Configures a Redux store for a spec that needs less than createTestStore, with
 * STORE_CHECKS as the default options of the getDefault passed to middleware.
 */
export const createMinimalStore: typeof configureStore = (options) =>
  configureStore({
    ...options,
    middleware: (getDefault) => {
      const getChecked = ((checks) =>
        getDefault({ ...STORE_CHECKS, ...checks })) as typeof getDefault;
      return options.middleware?.(getChecked) ?? (getChecked() as never);
    },
  });

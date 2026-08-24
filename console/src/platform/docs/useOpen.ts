// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

import { Runtime } from "@/platform/runtime";

export const URL = "https://docs.synnaxlabs.com/reference/console/get-started";

/** Opens the Synnax documentation in the user's browser. */
export const useOpen = (): (() => void) => {
  const handleError = Status.useErrorHandler();
  return useCallback(
    () =>
      handleError(async () => await Runtime.openExternal(URL), "Failed to open docs"),
    [handleError],
  );
};

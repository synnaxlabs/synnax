// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Flux, type Task } from "@synnaxlabs/pluto";
import { array } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Analytics } from "@/platform/analytics";

/** An `afterSuccess` for task commands. It reports each automation they start. */
export const useCaptureDeploy = (): ((
  params: Flux.AfterSuccessParams<Task.CommandParams>,
) => void) => {
  const { capture } = Analytics.use();
  return useCallback(
    ({ data }) =>
      array
        .toArray(data)
        .filter(({ type }) => type === "start")
        .forEach(() => capture("automation_deployed", {})),
    [capture],
  );
};

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";
import { type Form, Observe, Synnax } from "@synnaxlabs/pluto";
import { caseconv, type UnknownRecord } from "@synnaxlabs/x";
import { useState } from "react";
import { type z } from "zod";

interface ParserError {
  message: string;
  path: string;
}

interface ParserErrorsDetails extends UnknownRecord {
  errors?: ParserError[];
}

export const useObserveState = <
  T extends ParserErrorsDetails,
  S extends z.ZodType = z.ZodType,
  C extends z.ZodType = z.ZodType,
>(
  setStatus: Form.UseReturn<S>["setStatus"],
  clearStatuses: Form.UseReturn<C>["clearStatuses"],
  taskKey?: string,
  initialState?: task.State<T>,
): task.State<T> | undefined => {
  const client = Synnax.use();
  const [taskState, setTaskState] = useState(initialState);
  Observe.useListener({
    key: [taskKey],
    open: async () => await client?.hardware.tasks.openStateObserver<T>(),
    onChange: (state) => {
      if (state.task !== taskKey) return;
      setTaskState(state);
      if (state.variant !== "error") clearStatuses();
      else if (state.details?.errors != null)
        state.details.errors.forEach((e) => {
          const path = `config.${caseconv.snakeToCamel(e.path)}`;
          setStatus(path, { variant: "error", message: e.message });
        });
    },
  });
  return taskState;
};

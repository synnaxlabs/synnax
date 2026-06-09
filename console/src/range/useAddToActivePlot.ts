// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { lineplot } from "@synnaxlabs/client";
import { Ranger, Status, Synnax } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { LAYOUT_TYPE } from "@/lineplot/layout";
import { selectPendingUpload } from "@/lineplot/selectors";
import { add } from "@/range/slice";
import { fromClientRange } from "@/range/translate";
import { type RootState } from "@/store";

export const useAddToActivePlot = (): ((keys: string[]) => void) => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const store = useStore<RootState>();
  const client = Synnax.use();
  // Resolved at render and captured by the callback so an add targets the plot the
  // user had active when they triggered it.
  const active = Layout.useActiveResource();
  const { retrieve } = Ranger.useRetrieveObservableMultiple({
    onChange: useCallback(
      ({ data, variant, status }) => {
        if (variant !== "success") {
          if (variant === "error") addStatus(status);
          return;
        }
        if (active == null || active.type !== LAYOUT_TYPE || client == null) return;
        // A plot still staging a pendingUpload does not exist on the server yet,
        // so the dispatch below would fail with not found. Skip until
        // useAutoUpload lands it; the user can retry once the plot is created.
        if (selectPendingUpload(store.getState(), active.key) != null) return;
        store.dispatch(add({ ranges: fromClientRange(data) }));
        handleError(
          () =>
            client.lineplots.dispatch(
              active.key,
              id.create(),
              data.map((range) =>
                lineplot.addRange({ axisKey: "x1", range: range.key }),
              ),
            ),
          "Failed to add ranges to plot",
        );
      },
      [store, active, client, addStatus, handleError],
    ),
  });
  return useCallback((keys: string[]) => retrieve({ keys }), [retrieve]);
};

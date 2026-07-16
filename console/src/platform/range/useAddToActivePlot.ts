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

import { LinePlot } from "@/platform/lineplot";
import { Session } from "@/session";

export const useAddToActivePlot = (): ((keys: string[]) => void) => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const store = Session.useStore();
  const client = Synnax.use();
  const { retrieve } = Ranger.useRetrieveObservableMultiple({
    onChange: useCallback(
      ({ data, variant, status }) => {
        if (variant !== "success") {
          if (variant === "error") addStatus(status);
          return;
        }
        const active = Session.Layout.selectActiveMosaicLayout(store.getState());
        if (active == null || active.type !== LinePlot.LAYOUT_TYPE || client == null)
          return;
        store.dispatch(Session.Range.add(Session.Range.fromClient(data)));
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
      [store, client, addStatus, handleError],
    ),
  });
  return useCallback((keys: string[]) => retrieve({ keys }), []);
};

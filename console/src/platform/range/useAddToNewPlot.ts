// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Ranger, Status, Synnax } from "@synnaxlabs/pluto";
import { strings } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Layout } from "@/platform/layout";
import { create as createLinePlot } from "@/platform/lineplot/layout";
import { Session } from "@/session";
import { add } from "@/session/range/slice";

export const useAddToNewPlot = (): ((keys: string[]) => void) => {
  const addStatus = Status.useAdder();
  const handleError = Status.useErrorHandler();
  const store = Session.useStore();
  const placeLayout = Layout.usePlacer();
  const client = Synnax.use();
  const { retrieve } = Ranger.useRetrieveObservableMultiple({
    onChange: useCallback(
      ({ data, variant, status }) => {
        if (variant !== "success") {
          if (variant === "error") addStatus(status);
          return;
        }
        if (client == null) return;
        Session.Range.fromClient(data).forEach((r) => store.dispatch(add(r)));
        const names = data.map(({ name }) => name);
        const keys = data.map(({ key }) => key);
        const project = Session.Project.selectSelected(store.getState());
        handleError(async () => {
          const { key, name } = await client.lineplots.create(project, {
            name: `Plot for ${strings.naturalLanguageJoin(names, "range")}`,
            ranges: { x1: keys, x2: [] },
          });
          placeLayout(createLinePlot({ key, name }));
        }, "Failed to create plot");
      },
      [store, client, addStatus, handleError, placeLayout],
    ),
  });
  return useCallback((keys: string[]) => retrieve({ keys }), [retrieve]);
};

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

import { LinePlot } from "@/platform/lineplot";
import { Session } from "@/session";

export const useAddToNewPlot = (): ((keys: string[]) => void) => {
  const addStatus = Status.useAdder();
  const client = Synnax.use();
  const create = LinePlot.useCreate();
  const dispatch = Session.useDispatch();
  const { retrieve } = Ranger.useRetrieveObservableMultiple({
    onChange: useCallback(
      ({ data, variant, status }) => {
        if (variant !== "success") {
          if (variant === "error") addStatus(status);
          return;
        }
        if (client == null) return;
        dispatch(Session.Range.add(Session.Range.fromClient(data)));
        const names = data.map(({ name }) => name);
        const keys = data.map(({ key }) => key);
        create({
          name: `Plot for ${strings.naturalLanguageJoin(names, "range")}`,
          ranges: { x1: keys, x2: [] },
        });
      },
      [client, addStatus, create],
    ),
  });
  return useCallback((keys: string[]) => retrieve({ keys }), [retrieve]);
};

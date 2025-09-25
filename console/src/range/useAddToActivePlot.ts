// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Ranger, Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";
import { useStore } from "react-redux";

import { Layout } from "@/layout";
import { setRanges } from "@/lineplot/slice";
import { add } from "@/range/slice";
import { fromClientRange } from "@/range/translate";
import { type RootState } from "@/store";

export const useAddToActivePlot = (): ((keys: string[]) => void) => {
  const addStatus = Status.useAdder();
  const store = useStore<RootState>();
  const { retrieve } = Ranger.useRetrieveObservableMultiple({
    onChange: ({ data, variant, status }) => {
      if (variant !== "success") {
        if (variant === "error") addStatus(status);
        return;
      }
      const active = Layout.selectActiveMosaicLayout(store.getState());
      if (active == null) return;
      store.dispatch(add({ ranges: fromClientRange(data) }));
      store.dispatch(
        setRanges({
          key: active.key,
          axisKey: "x1",
          mode: "add",
          ranges: data.map((range) => range.key),
        }),
      );
    },
  });
  return useCallback((keys: string[]) => retrieve({ keys }), []);
};

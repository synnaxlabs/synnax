// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";
import { useDispatch } from "react-redux";

import { createMosaicWindow, moveMosaicTab } from "@/layout/slice";
import { usePlacer } from "@/layout/usePlacer";

interface OpenInNewWindow {
  (layoutKey: string): void;
}

export const useOpenInNewWindow = (): OpenInNewWindow => {
  const dispatch = useDispatch();
  const place = usePlacer();
  return useCallback(
    (layoutKey) => {
      const { key } = place(createMosaicWindow({}));
      dispatch(
        moveMosaicTab({ windowKey: key, key: 1, tabKey: layoutKey, loc: "center" }),
      );
    },
    [dispatch, place],
  );
};

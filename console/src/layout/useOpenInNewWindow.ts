// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useDispatch } from "react-redux";

import { usePlacer } from "@/layout/hooks";
import { createMosaicWindow, moveMosaicTab } from "@/layout/slice";

export const useOpenInNewWindow = () => {
  const dispatch = useDispatch();
  const place = usePlacer();
  return (layoutKey: string) => {
    const { key } = place(createMosaicWindow({}));
    dispatch(
      moveMosaicTab({ windowKey: key, key: 1, tabKey: layoutKey, loc: "center" }),
    );
  };
};

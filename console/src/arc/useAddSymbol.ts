// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch } from "@reduxjs/toolkit";
import { Arc, Theming } from "@synnaxlabs/pluto";
import { id, type xy } from "@synnaxlabs/x";
import { useCallback } from "react";

import { addElement } from "@/arc/slice";

export const useAddSymbol = (dispatch: Dispatch, layoutKey: string) => {
  const theme = Theming.use();
  return useCallback(
    (key: string, position?: xy.XY) => {
      const spec = Arc.Stage.REGISTRY[key];
      const initialProps = spec.defaultProps(theme);
      dispatch(
        addElement({
          key: layoutKey,
          elKey: id.create(),
          node: { zIndex: spec.zIndex, position },
          props: { key, ...initialProps },
        }),
      );
    },
    [dispatch, layoutKey],
  );
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";
import { id, type xy } from "@synnaxlabs/x";
import { useCallback } from "react";

export const useAddSymbol = (layoutKey: string) => {
  const addNode = Arc.useAddNode(layoutKey);
  return useCallback(
    (type: string, position?: xy.XY) => addNode({ key: id.create(), type, position }),
    [addNode],
  );
};

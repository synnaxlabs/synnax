// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { Drift } from "@synnaxlabs/drift";
import { id } from "@synnaxlabs/x";
import { useCallback } from "react";

import { Session } from "@/session";

export interface OpenWindow {
  (key?: panel.Key, props?: Omit<Drift.WindowProps, "key">): void;
}

/**
 * useOpenWindow opens a window showing the given panel, or the project's first panel
 * when none is given. A window is a viewport, so the panel is not moved or claimed: it
 * stays in the strip and any number of windows may show it at once.
 */
export const useOpenWindow = (): OpenWindow => {
  const dispatch = Session.useDispatch();
  return useCallback(
    (key, props) => {
      const windowKey = id.create();
      dispatch(Drift.createWindow({ ...props, key: windowKey }));
      if (key != null) dispatch(Session.Panel.select({ windowKey, key }));
    },
    [dispatch],
  );
};

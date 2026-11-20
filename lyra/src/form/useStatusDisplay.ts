// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { deep } from "@synnaxlabs/x";
import { useEffect } from "react";

import { useContext } from "@/form/Context";

/**
 * Tells the form that the caller shows the status at the path while it is mounted, so
 * {@link ContextValue.focus} finds it. A null path shows nothing.
 */
export const useStatusDisplay = (path: deep.Target<unknown> | null): void => {
  const { addStatusDisplay } = useContext(
    undefined,
    `useStatusDisplay(${String(path)})`,
  );
  useEffect(
    () => (path == null ? undefined : addStatusDisplay(path)),
    [addStatusDisplay, path],
  );
};

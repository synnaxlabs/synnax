// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type deep } from "@synnaxlabs/x";
import { useEffect } from "react";

import { useContext } from "@/form/Context";
import { useSyncedRef } from "@/hooks/ref";

/**
 * Runs the handler with the path that the form asks to take focus, when the handler is
 * the one that takes it, while the caller is mounted. See {@link ContextValue.focus}.
 */
export const useFocusHandler = (
  path: deep.Target<unknown>,
  handler: (path: string) => void,
): void => {
  const { addFocusHandler } = useContext(undefined, `useFocusHandler(${String(path)})`);
  const handlerRef = useSyncedRef(handler);
  useEffect(
    () => addFocusHandler(path, (target) => handlerRef.current(target)),
    [addFocusHandler, path, handlerRef],
  );
};

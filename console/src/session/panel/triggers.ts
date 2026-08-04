// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Modals } from "@/session/modals";
import { useGetTabIsFocused } from "@/session/panel/selectors";

/**
 * @returns a getter for whether the surrounding tab is the one the user is working in,
 * and so should answer keyboard triggers. False for a background tab, and false for
 * every tab while a modal is open.
 *
 * A getter rather than a value: it is read when a trigger fires, so the answer can
 * change without re-rendering its consumer.
 */
export const useGetTabTriggersActive = (): (() => boolean) => {
  const getIsFocused = useGetTabIsFocused();
  const modals = Modals.useStore("useGetTabTriggersActive");
  return useCallback(
    () => getIsFocused() && !modals.isAnyOpen(),
    [getIsFocused, modals],
  );
};

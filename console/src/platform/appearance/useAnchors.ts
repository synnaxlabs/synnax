// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { record } from "@synnaxlabs/x";
import { useState } from "react";

/** Sets or, with null, clears the element that the picker for the key opens under. */
export type SetAnchor = (key: string, anchor: HTMLElement | null) => void;

/**
 * Tracks the element each picker opens under, such as the tab of the item it styles.
 * Render each picker outside the tabs, so its events never reach a tab.
 *
 * @returns the anchors by key, and a setter for a tab's ref to call.
 */
export const useAnchors = (): [Record<string, HTMLElement>, SetAnchor] => {
  const [anchors, setAnchors] = useState<Record<string, HTMLElement>>({});
  const setAnchor: SetAnchor = (key, anchor) =>
    setAnchors((prev) =>
      anchor == null ? record.omit(prev, key) : { ...prev, [key]: anchor },
    );
  return [anchors, setAnchor];
};

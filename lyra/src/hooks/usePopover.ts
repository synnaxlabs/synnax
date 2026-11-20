// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type RefObject, useLayoutEffect } from "react";

/**
 * Shows the element in ref in the top layer while shown is true. The element must set
 * `popover`. It shows again, last in the top layer, each time key changes. Pass the
 * element it portals into as key, since a new container mounts a new element.
 */
export const usePopover = (
  ref: RefObject<HTMLElement | null>,
  shown: boolean,
  key: unknown,
): void =>
  useLayoutEffect(() => {
    const el = ref.current;
    if (!shown || el == null) return;
    el.showPopover();
    return () => el.hidePopover();
  }, [ref, shown, key]);

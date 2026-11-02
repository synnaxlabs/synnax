// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { capture } from "@synnaxlabs/site-common/analytics";

/**
 * Captures each click on an element with a `data-event` attribute, as that event, with
 * the element's `data-location` as the `location` property.
 */
export const captureClicks = (): void => {
  document.addEventListener("click", (e) => {
    if (!(e.target instanceof Element)) return;
    const el = e.target.closest<HTMLElement>("[data-event]");
    if (el == null) return;
    const { event, location } = el.dataset;
    if (event == null || location == null)
      throw new Error("every [data-event] element needs a data-location");
    capture(event, { location });
  });
};

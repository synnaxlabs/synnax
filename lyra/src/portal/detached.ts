// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// display: contents keeps the element out of layout entirely, so children size and
// position against the element that hosts it as if they were its direct children.
export const createDetachedElement = (): HTMLElement => {
  const el = document.createElement("div");
  el.style.display = "contents";
  return el;
};

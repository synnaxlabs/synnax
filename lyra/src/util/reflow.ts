// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// Trigger reflow causes the browser to re-paint the element. This is necessary to
// fix white-spacing and wrapping issues in safari when the text gets dynamically
// changed.
export const triggerReflow = (el: HTMLElement): void => {
  if (el == null) return;
  el.style.display = "none";
  // eslint-disable-next-line @typescript-eslint/no-unused-expressions
  el.offsetHeight;
  el.style.display = "";
};

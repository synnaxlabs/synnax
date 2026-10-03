// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Adds `header--top` to the header while the page is within 10 pixels of the top. */
export const start = (): void => {
  const header = document.querySelector(".header");
  const update = (): void => {
    header?.classList.toggle("header--top", window.scrollY <= 10);
  };
  update();
  window.addEventListener("scroll", update, { passive: true });
};

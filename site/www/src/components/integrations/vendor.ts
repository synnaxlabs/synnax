// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

const IDLE = "existing";

/** Names the vendor under the pointer in its brand color, in place of "existing". */
export const start = (): void => {
  const vendor = document.querySelector<HTMLElement>(".integrations-vendor");
  for (const cell of document.querySelectorAll(".integration-cell")) {
    cell.addEventListener("mouseenter", () => {
      const name = cell.getAttribute("data-vendor");
      if (vendor == null || name == null) return;
      vendor.textContent = name;
      vendor.style.color = getComputedStyle(cell).getPropertyValue("--brand-color");
    });
    cell.addEventListener("mouseleave", () => {
      if (vendor == null) return;
      vendor.textContent = IDLE;
      vendor.style.color = "";
    });
  }
};

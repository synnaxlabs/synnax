// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Names the vendor under the pointer in its brand color, in place of the idle text. */
export const start = (): void => {
  const vendor = document.querySelector<HTMLElement>(".integrations-vendor");
  if (vendor == null) throw new Error("the page has no .integrations-vendor");
  const idle = vendor.textContent;
  for (const cell of document.querySelectorAll(".integration-cell")) {
    const name = cell.getAttribute("data-vendor");
    if (name == null) throw new Error("an .integration-cell has no data-vendor");
    cell.addEventListener("mouseenter", () => {
      vendor.textContent = name;
      vendor.style.color = getComputedStyle(cell).getPropertyValue("--brand-color");
    });
    cell.addEventListener("mouseleave", () => {
      vendor.textContent = idle;
      vendor.style.color = "";
    });
  }
};

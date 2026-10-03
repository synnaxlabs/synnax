// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Shows the workflow panel of each tab when the tab is clicked. */
export const start = (): void => {
  const tabs = document.querySelectorAll(".workflow-tab");
  const panels = document.querySelectorAll(".workflow-panel");
  for (const tab of tabs)
    tab.addEventListener("click", () => {
      const step = tab.getAttribute("data-step");
      for (const t of tabs) t.classList.remove("workflow-tab--active");
      for (const p of panels) p.classList.remove("workflow-panel--active");
      tab.classList.add("workflow-tab--active");
      document
        .querySelector(`.workflow-panel[data-panel="${step}"]`)
        ?.classList.add("workflow-panel--active");
    });
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "zod/compile";
import "@/theming/base.css";

import { failOnConsoleOutput } from "@synnaxlabs/x/testutil";
import { afterEach } from "vitest";
import { userEvent } from "vitest/browser";

failOnConsoleOutput();

// Every host sets border-box on every element (console/src/app/App.css), and Lyra's
// layout depends on it. A spec reads a computed style at once, so a transition would
// leave it mid-way.
const host = document.createElement("style");
host.textContent = `
  * { box-sizing: border-box; }
  *, *::before, *::after { transition: none !important; animation: none !important; }
`;
document.head.appendChild(host);

// The pointer stays where a spec left it, so the next spec's element would render
// hovered.
afterEach(async () => {
  const corner = document.createElement("div");
  corner.style.cssText =
    "position: fixed; right: 0; bottom: 0; width: 1px; height: 1px";
  document.body.appendChild(corner);
  await userEvent.hover(corner, { force: true });
  corner.remove();
});

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "zod/compile";
import "@synnaxlabs/lyra/dist/base.css";
import "@synnaxlabs/pluto/dist/pluto.css";

import { CSS } from "@synnaxlabs/lyra/css";
import { setupBrowser } from "@synnaxlabs/lyra/styletest";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { failOnConsoleOutput } from "@synnaxlabs/x/testutil";
import { beforeEach, onTestFailed } from "vitest";

import { traceResizeLoops } from "@/testutil/resizeLoop";

failOnConsoleOutput();
// Vitest prints the console output of passed tests, so only a failed test prints the
// loop reports.
const loopReports: string[] = [];
traceResizeLoops((report) => loopReports.push(report));
beforeEach(() => {
  loopReports.length = 0;
  onTestFailed(() => loopReports.forEach((report) => console.info(report)));
});
setupBrowser();
// The Console sets its theme on the document root, where drawings such as the Nebula
// backdrop read their colors.
CSS.applyVars(
  document.documentElement,
  Theming.toCSSVars(theme.themeZ.parse(theme.SYNNAX_LIGHT)),
);

// Testing Library unmounts each spec in an afterEach hook, which must run before the
// pointer reset of setupBrowser. Hooks run last-in first-out, so it loads after.
const { configure, prettyDOM } = await import("@testing-library/react");
configure({
  // Waits allow 5 s, as in the jsdom setup.
  asyncUtilTimeout: 5000,
  // A browser has no DEBUG_PRINT_LIMIT, so a failed query prints the whole page.
  getElementError: (message, container) => {
    const dom = prettyDOM(container, Number.POSITIVE_INFINITY);
    const error = new Error(`${message ?? ""}\n\n${dom === false ? "" : dom}`);
    error.name = "TestingLibraryElementError";
    return error;
  },
});

// The Lyra testutil imports Testing Library, so it also loads after setupBrowser.
const { preloadTimeLanguage } = await import("@synnaxlabs/lyra/testutil");
preloadTimeLanguage();

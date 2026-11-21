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

import { CSS } from "@synnaxlabs/lyra/css";
import { setupBrowser } from "@synnaxlabs/lyra/styletest";
import { theme } from "@synnaxlabs/lyra/theme";
import { Theming } from "@synnaxlabs/lyra/theming";
import { failOnConsoleOutput } from "@synnaxlabs/x/testutil";

failOnConsoleOutput();
setupBrowser();
// The Console sets its theme on the document root, where drawings such as the Nebula
// backdrop read their colors.
CSS.applyVars(
  document.documentElement,
  Theming.toCSSVars(theme.themeZ.parse(theme.SYNNAX_LIGHT)),
);

// Testing Library unmounts each spec in an afterEach hook, which must run before the
// pointer reset of setupBrowser. Hooks run last-in first-out, so it loads after.
const { configure } = await import("@testing-library/react");
// Waits allow 5 s, as in the jsdom setup.
configure({ asyncUtilTimeout: 5000 });

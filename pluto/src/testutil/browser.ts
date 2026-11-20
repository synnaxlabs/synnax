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

import { setupBrowser } from "@synnaxlabs/lyra/styletest";
import { failOnConsoleOutput } from "@synnaxlabs/x/testutil";

// Monaco warns when a startup step is slow, which depends on the machine.
failOnConsoleOutput({ ignore: [/Creation of workbench contribution '.+' took \d+ms/] });
setupBrowser();

// Testing Library unmounts each spec in an afterEach hook, which must run before the
// pointer reset of setupBrowser. Hooks run last-in first-out, so it loads after.
const { configure } = await import("@testing-library/react");
// Waits allow 5 s, as in the jsdom setup.
configure({ asyncUtilTimeout: 5000 });

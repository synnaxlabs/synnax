// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "zod/compile";

import { createRoot } from "react-dom/client";

import { App } from "@/app";
import { mountUIOptionsBar } from "@/uiOptionsBar";
import workerURL from "@/worker?worker&url";

const rootEl = document.getElementById("root") as HTMLElement;

mountUIOptionsBar();

createRoot(rootEl).render(<App.App workerURL={workerURL} />);

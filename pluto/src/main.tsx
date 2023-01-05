// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { StrictMode } from "react";

import { createRoot } from "react-dom/client";

import { App } from "./App";

const root = document.getElementById("root") as HTMLElement;
createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>
);

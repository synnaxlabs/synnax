// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { createRoot } from "react-dom/client";

import { Console } from "@/Console";

const rootEl = document.getElementById("root") as HTMLElement;

createRoot(rootEl).render(<Console />);

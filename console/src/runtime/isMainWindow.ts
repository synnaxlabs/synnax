// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Drift } from "@synnaxlabs/drift";
import { getCurrentWindow } from "@tauri-apps/api/window";

import { ENGINE } from "@/runtime/runtime";

export const isMainWindow = (): boolean => {
  switch (ENGINE) {
    case "tauri":
      return getCurrentWindow().label === Drift.MAIN_WINDOW;
    case "web":
      return true;
  }
};

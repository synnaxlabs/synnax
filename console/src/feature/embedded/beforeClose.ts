// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { MAIN_WINDOW } from "@synnaxlabs/drift";

import { quit } from "@/feature/embedded/supervisor";

/**
 * Quits the app when the main window closes. The window stays open while the embedded
 * Core stops, so it shows the stop.
 */
export const beforeClose = async (label: string): Promise<void> => {
  if (label === MAIN_WINDOW) await quit();
};

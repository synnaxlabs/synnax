// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { getVersion } from "@tauri-apps/api/app";

// A dev build runs at 0.0.0 and has no release to update to.
export const isDevBuild = async (): Promise<boolean> =>
  (await getVersion()).startsWith("0.0.");

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { openProject } from "./common/desktop";
import { connectServerScript } from "./common/server";
import { SIFT } from "./common/sift";

export const desktop = true;

/**
 * Docs `reference/ranges/sift` (Connect Sift): connect the fake Sift server from the
 * devices toolbar, with its API key saved as a new secret, from the rack of the Core.
 */
export default connectServerScript(SIFT, openProject);

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Recorder } from "@/vis/render/test/Recorder";

/** Construct a fresh canvas {@link Recorder}. Returns an object that can be passed as
 * the `render` option to `renderAether` (or `render`), or used standalone to record
 * draw calls from non-aether code. */
export const record = (): Recorder => new Recorder();

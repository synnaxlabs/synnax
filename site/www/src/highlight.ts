// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { grammar } from "@synnaxlabs/arc";
import { highlighter } from "@synnaxlabs/site-common/shiki";

/** Renders code in Python, TypeScript, or Arc to highlighted HTML. */
export const highlight = await highlighter(["python", "typescript", grammar]);

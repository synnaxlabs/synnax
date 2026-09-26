// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@/context";

/** Marks the subtree of a Scroll, so an Items element can require one. */
export const [ScrollContext, useScrollContext] = context.create<true>({
  displayName: "List.ScrollContext",
  providerName: "List.Scroll",
});

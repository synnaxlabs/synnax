// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import type { Theming } from "@synnaxlabs/lyra/theming";

/**
 * The background of a page's section list and cards. A page on an l1 surface sets it to
 * 0, so the cards stand off the surface.
 */
const [CardBackgroundContext, useCardBackground] = context.create<Theming.Shade>({
  defaultValue: 1,
  displayName: "Page.CardBackgroundContext",
});

export { CardBackgroundContext, useCardBackground };

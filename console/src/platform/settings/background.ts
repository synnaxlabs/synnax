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
 * The background of the Settings nav and cards. Settings on an l1 page sets it to 0, so
 * the cards stand off the page.
 */
const [CardBackgroundContext, useCardBackground] = context.create<Theming.Shade>({
  defaultValue: 1,
  displayName: "Settings.CardBackgroundContext",
});

export { CardBackgroundContext, useCardBackground };

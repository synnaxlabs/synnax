// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type panel } from "@synnaxlabs/client";
import { uuid } from "@synnaxlabs/x";

// SELECTOR_VIEW_TYPE is the view type of the component selector. The selector is a
// regular view renderer (registered in LAYOUT_RENDERERS) that, when a user picks a
// component, replaces its own tab with the chosen resource or view.
export const SELECTOR_VIEW_TYPE = "selector";

// selectorTab produces a fresh selector view tab. The console seeds it into leaves
// that would otherwise be empty (a new panel, a new tab, or splitting one off),
// landing the user in the component picker.
export const selectorTab = (): panel.Tab => ({
  key: uuid.create(),
  type: SELECTOR_VIEW_TYPE,
  args: {},
});

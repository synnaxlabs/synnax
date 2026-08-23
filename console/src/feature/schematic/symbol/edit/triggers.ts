// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Triggers } from "@synnaxlabs/pluto";

export const ZOOM_TRIGGERS: Triggers.ModeConfig<"in" | "out" | "reset" | "default"> = {
  defaultMode: "default",
  modes: {
    in: [["Control", "Equal"]],
    out: [["Control", "Minus"]],
    reset: [["Control", "0"]],
    default: [],
  },
};

export const FLATTENED_ZOOM_TRIGGERS = Triggers.flattenConfig(ZOOM_TRIGGERS);

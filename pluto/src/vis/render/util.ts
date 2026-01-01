// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, type xy } from "@synnaxlabs/x";

export const applyOverScan = (b: box.Box, overScan: xy.XY): box.Box =>
  box.construct(
    box.left(b) - overScan.x,
    box.top(b) - overScan.y,
    box.width(b) + overScan.x * 2,
    box.height(b) + overScan.y * 2,
  );

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot } from "@synnaxlabs/pluto";

import { create } from "@/lineplot/layout";
import { Range } from "@/range";
import { Workspace } from "@/workspace";

export const useCreate = Workspace.createUseCreate({
  useCreate: LinePlot.useCreate,
  createLayout: ({ key, name }) => create({ key, name }),
  defaultName: "Line Plot",
  useDefaults: () => {
    const activeRange = Range.useSelectActiveKey() ?? Range.RECENT_RANGE_KEY;
    return { ranges: { x1: [activeRange], x2: [] } };
  },
});

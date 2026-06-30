// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { LinePlot } from "@synnaxlabs/pluto";

import { create } from "@/service/lineplot/layout";
import { Project } from "@/project";
import { Range } from "@/range";

export const useCreate = Project.createUseCreate({
  useCreate: LinePlot.useCreate,
  toCreateParams: ({ overrides, project, store }) => {
    const activeRange = Range.selectSelectedKey(store.getState()) ?? Range.RECENT_KEY;
    return {
      name: "Line Plot",
      ranges: { x1: [activeRange] },
      ...overrides,
      project,
    };
  },
  createSessionState: create,
});

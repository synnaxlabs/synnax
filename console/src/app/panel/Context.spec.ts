// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  arc,
  icd,
  lineplot,
  log,
  ranger,
  type resource,
  schematic,
  table,
  task,
} from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { uuid } from "@synnaxlabs/x";
import { renderHook } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { Panel } from "@/app/panel";
import { Panel as PlatformPanel } from "@/platform/panel";

interface Case {
  id: resource.ID;
  icon: string;
}

const CASES: Case[] = [
  { id: lineplot.TYPE_RESOURCE_ID, icon: "line-plot" },
  { id: schematic.TYPE_RESOURCE_ID, icon: "schematic" },
  { id: table.TYPE_RESOURCE_ID, icon: "table" },
  { id: log.TYPE_RESOURCE_ID, icon: "log" },
  { id: task.TYPE_RESOURCE_ID, icon: "task" },
  { id: arc.TYPE_RESOURCE_ID, icon: "arc" },
  { id: ranger.TYPE_RESOURCE_ID, icon: "range" },
  { id: icd.TYPE_RESOURCE_ID, icon: "library" },
];

describe("Panel.Context", () => {
  it.each(CASES)(
    "should give a panel for a $id.type the $icon icon",
    ({ id, icon }) => {
      const { result } = renderHook(() => PlatformPanel.useGetIconKey(), {
        wrapper: Panel.Context,
      });
      const key = result.current({
        variant: "resource",
        key: uuid.create(),
        resource: id,
      });
      expect(key).toEqual(icon);
      expect(Icon.getPickable(icon)).toBeDefined();
    },
  );
});

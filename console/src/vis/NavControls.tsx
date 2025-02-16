// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Align } from "@synnaxlabs/pluto";
import { type FC, type ReactElement } from "react";

import { CSS } from "@/css";
import { Layout } from "@/layout";
import { LinePlot } from "@/lineplot";
import { NavControls as LineNavControls } from "@/lineplot/NavControls";
import { Log } from "@/log";
import { Schematic } from "@/schematic";
import { NavControls as SchematicNavControls } from "@/schematic/NavControls";
import { Table } from "@/table";
import { type LayoutType } from "@/vis/types";

const REGISTRY: Record<LayoutType, FC> = {
  [LinePlot.LAYOUT_TYPE]: LineNavControls,
  [Log.LAYOUT_TYPE]: () => null,
  [Schematic.LAYOUT_TYPE]: SchematicNavControls,
  [Table.LAYOUT_TYPE]: () => null,
};

export const NavControls = (): ReactElement | null => {
  const layout = Layout.useSelectActiveMosaicLayout();
  if (layout == null) return null;
  const Controls = REGISTRY[layout.type as LayoutType];
  return Controls == null ? null : (
    <Align.Space direction="x" empty className={CSS.B("nav-controls")}>
      <Controls />
    </Align.Space>
  );
};

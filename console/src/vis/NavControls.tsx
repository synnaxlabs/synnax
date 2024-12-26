// Copyright 2024 Synnax Labs, Inc.
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
import { NavControls as LineNavControls } from "@/lineplot/NavControls";
import { NavControls as SchematicNavControls } from "@/schematic/NavControls";
import { type LayoutType } from "@/vis/types";

const REGISTRY: Record<LayoutType, FC> = {
  schematic: SchematicNavControls,
  lineplot: LineNavControls,
  log: () => null,
  table: () => null,
};

export const NavControls = (): ReactElement | null => {
  const layout = Layout.useSelectActiveMosaicLayout();
  if (layout == null) return null;

  const Controls = REGISTRY[layout.type as LayoutType];
  if (Controls == null) return null;

  return (
    <Align.Space direction="x" empty className={CSS.B("nav-controls")}>
      <Controls />
    </Align.Space>
  );
};

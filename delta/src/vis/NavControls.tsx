// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type FC, type ReactElement } from "react";

import { Align } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { type LinePlot } from "@/lineplot";
import { NavControls as LineNavControls } from "@/lineplot/NavControls";
import { type PID } from "@/pid";
import { NavControls as PidNavControls } from "@/pid/NavControls";

const REGISTRY: Record<Type | Type, FC> = {
  lineplot: LineNavControls,
  pid: PidNavControls,
};

export const NavControls = (): ReactElement | null => {
  const layout = Layout.useSelectActiveMosaicLayout();
  if (layout == null) return null;

  const Controls = REGISTRY[layout.type as Type];
  if (Controls == null) return null;

  return (
    <Align.Space direction="x" size="small">
      <Controls />
    </Align.Space>
  );
};

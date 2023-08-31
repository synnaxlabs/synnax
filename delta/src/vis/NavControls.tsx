// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FC, ReactElement } from "react";

import { Align } from "@synnaxlabs/pluto";

import { Layout } from "@/layout";
import { NavControls as LineNavControls } from "@/line/NavControls";
import { NavControls as PidNavControls } from "@/pid/NavControls";

import { ViewportModeSelector } from "./ViewportModeSelector";

const REGISTRY: Record<"line" | "pid", FC> = {
  line: LineNavControls,
  pid: PidNavControls,
};

export const NavControls = (): ReactElement => {
  const layout = Layout.useSelectActiveMosaicLayout();

  const Controls = REGISTRY[layout?.type as "line" | "pid"] ?? (() => <></>);

  return (
    <Align.Space direction="x" size="small">
      <ViewportModeSelector />
      <Controls />
    </Align.Space>
  );
};

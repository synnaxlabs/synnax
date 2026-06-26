// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";
import { type ReactElement } from "react";

import { Toolbar as GraphToolbar } from "@/layered/service/arc/editor/toolbar/graph/Toolbar";
import { Toolbar as TextToolbar } from "@/layered/service/arc/editor/toolbar/text/Toolbar";

export interface ToolbarProps {
  layoutKey: string;
}

const Internal = (): ReactElement | null => {
  const mode = Arc.useSelectMode();
  if (mode === "text") return <TextToolbar />;
  return <GraphToolbar />;
};

export const Toolbar = ({ layoutKey }: ToolbarProps): ReactElement => (
  <Arc.Suspended arcKey={layoutKey}>
    <Internal />
  </Arc.Suspended>
);

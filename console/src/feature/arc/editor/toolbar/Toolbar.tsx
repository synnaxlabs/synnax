// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc, Panel as PPanel } from "@synnaxlabs/pluto";

import { Graph } from "@/feature/arc/editor/toolbar/graph";
import { Text } from "@/feature/arc/editor/toolbar/text";
import { type Panel } from "@/platform/panel";

const Internal = () => {
  const mode = Arc.useSelectMode();
  return mode === "text" ? <Text.Toolbar /> : <Graph.Toolbar />;
};

export const Toolbar: Panel.Toolbar = () => {
  const { key } = PPanel.useSelectTabResource();
  return (
    <Arc.Suspended arcKey={key}>
      <Internal />
    </Arc.Suspended>
  );
};

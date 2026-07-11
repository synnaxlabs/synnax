// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc, Panel as PPanel } from "@synnaxlabs/pluto";

import { Graph } from "@/feature/arc/editor/Graph";
import { Text } from "@/feature/arc/editor/Text";
import { type Panel } from "@/platform/panel";

export const Editor: Panel.Content = () => {
  const { key } = PPanel.useSelectTabResource();
  const mode = Arc.useSelectMode({ key });
  return (
    <Arc.Suspended arcKey={key}>
      {mode === "graph" ? <Graph /> : <Text />}
    </Arc.Suspended>
  );
};

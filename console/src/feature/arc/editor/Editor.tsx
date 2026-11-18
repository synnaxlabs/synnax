// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/arc/editor/Editor.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Arc, Panel as PPanel } from "@synnaxlabs/pluto";

import { Graph } from "@/feature/arc/editor/Graph";
import { Path } from "@/feature/arc/editor/Path";
import { Text } from "@/feature/arc/editor/Text";
import { CSS } from "@/platform/css";
import { type Panel } from "@/platform/panel";

const Internal = () => {
  const mode = Arc.useMode();
  return mode === "graph" ? <Graph /> : <Text />;
};

export const Editor: Panel.Content = () => {
  const { key } = PPanel.useTabResource();
  return (
    <Arc.Suspended arcKey={key}>
      <Flex.Box y empty full>
        <Path />
        <Flex.Box y empty grow className={CSS.BE("arc-editor", "body")}>
          <Internal />
        </Flex.Box>
      </Flex.Box>
    </Arc.Suspended>
  );
};

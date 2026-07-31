// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/range/overview/Overview.css";

import { Flex, Panel as PlutoPanel } from "@synnaxlabs/pluto";

import { ChildRanges } from "@/feature/range/overview/ChildRanges";
import { Snapshots } from "@/feature/range/overview/Snapshots";
import { CSS } from "@/platform/css";
import { type Panel } from "@/platform/panel";
import { Range } from "@/platform/range";

export const Overview: Panel.Content = () => {
  const { key } = PlutoPanel.useSelectTabResource();
  return (
    <Flex.Box key={key} y className={CSS.BE("range", "overview")} empty>
      <Flex.Box y grow gap="large">
        <Range.Details rangeKey={key} />
        <ChildRanges rangeKey={key} />
        <Range.MetaData rangeKey={key} />
        <Snapshots rangeKey={key} />
      </Flex.Box>
    </Flex.Box>
  );
};

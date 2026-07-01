// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/service/range/overview/Overview.css";

import { Flex, Ranger } from "@synnaxlabs/pluto";

import { CSS } from "@/component/css";
import { Layout } from "@/component/layout";
import { Range } from "@/component/range";
import { ChildRanges } from "@/service/range/overview/ChildRanges";
import { Snapshots } from "@/service/range/overview/Snapshots";

export const Overview: Layout.Renderer = ({ layoutKey }) => (
  <Flex.Box
    y
    style={{ padding: "5rem", maxWidth: 1050, margin: "0 auto", overflowY: "auto" }}
    className={CSS.BE("range", "overview")}
    gap="large"
  >
    <Range.Details rangeKey={layoutKey} />
    <ChildRanges rangeKey={layoutKey} />
    <Range.MetaData rangeKey={layoutKey} />
    <Snapshots rangeKey={layoutKey} />
  </Flex.Box>
);

Overview.useName = Layout.createUseFluxName(
  Ranger.useRename,
  Ranger.useRetrieveObservableName,
);

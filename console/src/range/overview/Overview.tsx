// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/range/overview/Overview.css";

import { Align, Divider } from "@synnaxlabs/pluto";

import { CSS } from "@/css";
import { type Layout } from "@/layout";
import { ChildRanges } from "@/range/overview/ChildRanges";
import { Details } from "@/range/overview/Details";
import { Labels } from "@/range/overview/Labels";
import { MetaData } from "@/range/overview/MetaData";
import { Snapshots } from "@/range/overview/Snapshots";

export const Overview: Layout.Renderer = ({ layoutKey }) => (
  <Align.Space
    direction="y"
    style={{
      padding: "5rem",
      maxWidth: 1050,
      margin: "0 auto",
      overflowY: "auto",
    }}
    className={CSS.BE("range", "overview")}
    size="medium"
  >
    <Details rangeKey={layoutKey} />
    <Labels rangeKey={layoutKey} />
    <Divider.Divider direction="x" />
    <MetaData rangeKey={layoutKey} />
    <Divider.Divider direction="x" />
    <ChildRanges rangeKey={layoutKey} />
    <Divider.Divider direction="x" />
    <Snapshots rangeKey={layoutKey} />
  </Align.Space>
);

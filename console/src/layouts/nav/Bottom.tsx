// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/layouts/nav/Nav.css";

import { Divider, Nav, Text } from "@synnaxlabs/pluto";
import { Size } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

import { Cluster } from "@/cluster";
import { CSS } from "@/css";
import { SIZES } from "@/layouts/nav/sizes";
import { Version } from "@/version";
import { Vis } from "@/vis";

interface MemoryUsage {
  used: Size;
  total: Size;
}

interface MemoryInfo {
  usedJSHeapSize: number;
  jsHeapSizeLimit: number;
}

interface PerformanceAPI {
  memory: MemoryInfo;
}

const MemoryBadge = () => {
  const [memory, setMemory] = useState<MemoryUsage>({
    used: Size.ZERO,
    total: Size.ZERO,
  });
  const hasMemoryDisplayed = "memory" in performance;
  useEffect(() => {
    const interval = setInterval(() => {
      if ("memory" in performance) {
        const { memory } = performance as PerformanceAPI;
        setMemory({
          used: Size.bytes(memory.usedJSHeapSize),
          total: Size.bytes(memory.jsHeapSizeLimit),
        });
      }
    }, 1000);
    return () => clearInterval(interval);
  });
  return !hasMemoryDisplayed ? null : (
    <>
      <Divider.Divider />
      <Text.Text level="p" style={{ padding: "0 2rem" }}>
        {memory.used.truncate(Size.MEGABYTE).toString()} /
        {memory.total.truncate(Size.MEGABYTE).toString()}
      </Text.Text>
    </>
  );
};

export const Bottom = () => (
  <Nav.Bar className={CSS.B("main-nav")} location="bottom" size={SIZES.bottom}>
    <Nav.Bar.Start>
      <Vis.NavControls />
    </Nav.Bar.Start>
    <Nav.Bar.End className="console-main-nav-bottom__end" empty>
      <MemoryBadge />
      <Divider.Divider />
      <Version.Badge />
      <Divider.Divider />
      <Cluster.Dropdown />
      <Divider.Divider />
      <Cluster.ConnectionBadge />
    </Nav.Bar.End>
  </Nav.Bar>
);

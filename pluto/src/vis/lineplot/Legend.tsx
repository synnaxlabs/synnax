// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/vis/legend/Container.css";

import { type ReactElement } from "react";

import { Legend as Core } from "@/vis/legend";
import { type LineSpec, useContext } from "@/vis/lineplot/LinePlot";

export interface LegendProps extends Omit<Core.SimpleProps, "data" | "onEntryChange"> {
  onLineChange?: (line: LineSpec) => void;
}

export const Legend = ({
  className,
  style,
  onLineChange,
  ...props
}: LegendProps): ReactElement | null => {
  const { lines } = useContext("Legend");
  useContext("Legend");
  return <Core.Simple data={lines} onEntryChange={onLineChange} {...props} />;
};

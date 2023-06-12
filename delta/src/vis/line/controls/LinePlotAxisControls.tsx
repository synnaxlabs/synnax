// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, useMemo } from "react";

import { Input, List } from "@synnaxlabs/pluto";

import { Axes } from "../axes";
import { Bounds } from "../bounds";
import { AxisState, BoundState } from "../LinePlot/core";

import { AxisKey } from "@/vis/axis";

export interface LinePlotAxisControlsProps {
  layoutKey: string;
}

export interface AxisListEntry extends BoundState, AxisState {
  key: string;
}

export const LinePlotAxisControls = ({
  layoutKey,
}: LinePlotAxisControlsProps): ReactElement => {
  const bounds = Bounds.useSelect(layoutKey);
  const axes = Axes.useSelect(layoutKey);
  const entries = useMemo(
    () =>
      (Object.entries(axes) as Array<[AxisKey, AxisState]>).map(([key, axis]) => ({
        ...bounds[key],
        ...axis,
        key: key.toUpperCase(),
      })),
    [axes, bounds]
  );

  return (
    <List<AxisListEntry> data={entries}>
      <List.Column.Header<AxisListEntry>
        columns={[
          {
            key: "key",
            name: "Axis",
          },
          {
            key: "name",
            name: "Name",
            render: ({ entry: { name }, style }) => (
              <Input value={name} style={style} onChange={console.log} />
            ),
            width: 100,
          },
          {
            key: "lowerBound",
            name: "Lower Bound",
            render: ({ entry: { bound }, style }) => (
              <Input.Number
                value={bound.lower}
                style={{ ...style, width: (style.width = 50) }}
                onChange={console.log}
              />
            ),
            width: 150,
          },
          {
            key: "upperBound",
            name: "Upper Bound",
            render: ({ entry: { bound }, style }) => (
              <Input.Number value={bound.upper} style={style} onChange={console.log} />
            ),
            width: 150,
          },
        ]}
      />
      <List.Core.Virtual itemHeight={30} style={{ height: "100%" }}>
        {List.Column.Item}
      </List.Core.Virtual>
    </List>
  );
};

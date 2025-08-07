// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Channel, Color, Flex, Input, List, Tabs } from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { EmptyAction } from "@/components";
import { useSelectLine, useSelectLineKeys } from "@/lineplot/selectors";
import { type LineState, setLine, typedLineKeyFromString } from "@/lineplot/slice";

export interface LinesProps {
  layoutKey: string;
}

export const Lines = ({ layoutKey }: LinesProps): ReactElement => {
  const lineKeys = useSelectLineKeys(layoutKey);
  const dispatch = useDispatch();

  const handleChange = (line: LineState): void => {
    dispatch(setLine({ key: layoutKey, line }));
  };

  const { onSelect } = Tabs.useContext();

  const emptyContent = (
    <EmptyAction
      x
      message="No lines plotted. Select channels using the"
      action="data tab."
      onClick={(e) => {
        e.stopPropagation();
        onSelect?.("data");
      }}
    />
  );

  return (
    <List.Frame data={lineKeys}>
      <List.Items<string, LineState>
        style={{ height: "calc(100% - 28px)" }}
        emptyContent={emptyContent}
      >
        {(p) => <Line layoutKey={layoutKey} onChange={handleChange} {...p} />}
      </List.Items>
    </List.Frame>
  );
};

interface LinePlotLineControlsProps extends Omit<List.ItemProps<string>, "onChange"> {
  layoutKey: string;
  onChange: (line: LineState) => void;
}

const Line = ({
  itemKey,
  onChange,
  layoutKey,
}: LinePlotLineControlsProps): ReactElement | null => {
  const line = useSelectLine(layoutKey, itemKey);
  if (line == null) return null;
  const handleLabelChange: Input.Control<string>["onChange"] = (value: string) => {
    onChange({ ...line, label: value });
  };

  const handleWidthChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...line, strokeWidth: value });
  };

  const handleDownsampleChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...line, downsample: value });
  };

  const handleColorChange: Input.Control<color.Color>["onChange"] = (
    value: color.Color,
  ) => {
    onChange({ ...line, color: color.hex(value) });
  };

  const {
    channels: { y: yChannel },
  } = typedLineKeyFromString(line.key);

  return (
    <Flex.Box style={{ padding: "0.5rem" }} x full="x">
      <Channel.AliasInput
        channelKey={yChannel}
        style={{ width: 305 }}
        value={line.label ?? ""}
        onChange={handleLabelChange}
      />
      <Input.Numeric
        value={line.strokeWidth}
        onChange={handleWidthChange}
        dragScale={{ x: 0.1, y: 0.1 }}
        bounds={{ lower: 1, upper: 11 }}
        style={{ width: 140, marginRight: "2rem" }}
      />
      <Input.Numeric
        style={{ width: 100, marginRight: "2rem" }}
        value={line.downsample ?? 1}
        onChange={handleDownsampleChange}
        dragScale={{ x: 0.1, y: 0.1 }}
        bounds={{ lower: 1, upper: 51 }}
      />
      <Color.Swatch value={line.color} onChange={handleColorChange} size="small" />
    </Flex.Box>
  );
};

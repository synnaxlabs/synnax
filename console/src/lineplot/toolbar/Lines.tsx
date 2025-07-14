// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  Align,
  Channel,
  Color,
  Input,
  List,
  Status,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { color } from "@synnaxlabs/x";
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

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
    <Align.Center x size="small">
      <Status.Text variant="disabled" hideIcon>
        No lines plotted. Use the
      </Status.Text>
      <Text.Link
        onClick={(e) => {
          e.stopPropagation();
          onSelect?.("data");
        }}
        level="p"
      >
        data
      </Text.Link>
      <Status.Text variant="disabled" hideIcon>
        tab to select channels on an axis.
      </Status.Text>
    </Align.Center>
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
    <Align.Space style={{ padding: "0.5rem", width: "100%" }} x>
      <Channel.AliasInput
        channelKey={yChannel}
        style={{ width: 305 }}
        value={line.label ?? ""}
        onChange={handleLabelChange}
        variant="shadow"
      />
      <Input.Numeric
        value={line.strokeWidth}
        onChange={handleWidthChange}
        dragScale={{ x: 0.1, y: 0.1 }}
        bounds={{ lower: 1, upper: 11 }}
        style={{ width: 140, marginRight: "2rem" }}
        variant="shadow"
      />
      <Input.Numeric
        style={{ width: 100, marginRight: "2rem" }}
        value={line.downsample ?? 1}
        onChange={handleDownsampleChange}
        variant="shadow"
        dragScale={{
          x: 0.1,
          y: 0.1,
        }}
        bounds={{
          lower: 1,
          upper: 51,
        }}
      />
      <Color.Swatch value={line.color} onChange={handleColorChange} size="small" />
    </Align.Space>
  );
};

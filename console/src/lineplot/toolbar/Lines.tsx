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
import { type ReactElement } from "react";
import { useDispatch } from "react-redux";

import { useSelect } from "@/lineplot/selectors";
import { type LineState, setLine, typedLineKeyFromString } from "@/lineplot/slice";

export interface LinesProps {
  layoutKey: string;
}

export const Lines = ({ layoutKey }: LinesProps): ReactElement => {
  const vis = useSelect(layoutKey);
  const dispatch = useDispatch();

  const handleChange = (line: LineState): void => {
    dispatch(setLine({ key: layoutKey, line }));
  };

  const { onSelect } = Tabs.useContext();

  const emptyContent = (
    <Align.Center direction="x" size="small">
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
    <List.List data={vis.lines} emptyContent={emptyContent}>
      <List.Column.Header
        columns={[
          { key: "label", name: "Label", width: 305 },
          { key: "strokeWidth", name: "Line Width", width: 150 },
          { key: "downsample", name: "Downsampling", width: 110 },
          { key: "color", name: "Color", width: 100 },
        ]}
      />
      <List.Core<string, LineState> style={{ height: "calc(100% - 28px)" }}>
        {(p) => <Line onChange={handleChange} {...p} />}
      </List.Core>
    </List.List>
  );
};

interface LinePlotLineControlsProps extends List.ItemProps<string, LineState> {
  onChange: (line: LineState) => void;
}

const Line = ({ entry, onChange }: LinePlotLineControlsProps): ReactElement => {
  const handleLabelChange: Input.Control<string>["onChange"] = (value: string) => {
    onChange({ ...entry, label: value });
  };

  const handleWidthChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...entry, strokeWidth: value });
  };

  const handleDownsampleChange: Input.Control<number>["onChange"] = (value: number) => {
    onChange({ ...entry, downsample: value });
  };

  const handleColorChange: Input.Control<Color.Color>["onChange"] = (
    value: Color.Color,
  ) => {
    onChange({ ...entry, color: value.hex });
  };

  const {
    channels: { y: yChannel },
  } = typedLineKeyFromString(entry.key);

  return (
    <Align.Space style={{ padding: "0.5rem", width: "100%" }} direction="x">
      <Channel.AliasInput
        channelKey={yChannel}
        style={{ width: 305 }}
        value={entry.label ?? ""}
        onChange={handleLabelChange}
        variant="shadow"
      />
      <Input.Numeric
        value={entry.strokeWidth}
        onChange={handleWidthChange}
        dragScale={{ x: 0.1, y: 0.1 }}
        bounds={{ lower: 1, upper: 11 }}
        style={{ width: 140, marginRight: "2rem" }}
        variant="shadow"
      />
      <Input.Numeric
        style={{ width: 100, marginRight: "2rem" }}
        value={entry.downsample ?? 1}
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
      <Color.Swatch
        value={new Color.Color(entry.color)}
        onChange={handleColorChange}
        size="small"
      />
    </Align.Space>
  );
};

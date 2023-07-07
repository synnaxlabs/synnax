// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement } from "react";

import {
  Input,
  InputControl,
  List,
  ListItemProps,
  Space,
  Status,
  Tabs,
  Text,
} from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";

import { useSelectLinePlot } from "../store/selectors";
import { LineState, setLinePlotLine } from "../store/slice";

export interface LinePlotLinesControlsProps {
  layoutKey: string;
}

export const LinePlotLinesControls = ({
  layoutKey,
}: LinePlotLinesControlsProps): ReactElement => {
  const vis = useSelectLinePlot(layoutKey);
  const dispatch = useDispatch();

  const handleChange = (line: LineState): void => {
    dispatch(setLinePlotLine({ key: layoutKey, line }));
  };

  const { onSelect } = Tabs.useContext();

  const emptyContent = (
    <Space.Centered direction="x" size="small">
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
    </Space.Centered>
  );

  return (
    <List data={vis.lines} emptyContent={emptyContent}>
      <List.Column.Header
        columns={[
          {
            key: "label",
            name: "Label",
            width: 300,
          },
          {
            key: "strokeWidth",
            name: "Line Width",
            width: 140,
          },
          {
            key: "downsample",
            name: "Downsampling",
            width: 100,
          },
        ]}
      />
      <List.Core.Virtual itemHeight={33} style={{ height: "calc(100% - 28px - 33px)" }}>
        {(props) => <LinePlotLineControls onChange={handleChange} {...props} />}
      </List.Core.Virtual>
    </List>
  );
};

interface LinePlotLineControlsProps extends ListItemProps<string, LineState> {
  line: LineState;
  onChange: (line: LineState) => void;
}

const LinePlotLineControls = ({
  entry: line,
  onChange,
}: LinePlotLineControlsProps): ReactElement => {
  const handleLabelChange: InputControl<string>["onChange"] = (value: string) => {
    onChange({ ...line, label: value });
  };

  const handleWidthChange: InputControl<number>["onChange"] = (value: number) => {
    onChange({ ...line, strokeWidth: value });
  };

  const handleDownsampleChange: InputControl<number>["onChange"] = (value: number) => {
    onChange({ ...line, downsample: value });
  };

  return (
    <Space style={{ padding: "0.5rem", width: "100%" }} direction="x">
      <Input
        style={{ width: 300 }}
        value={line.label ?? ""}
        onChange={handleLabelChange}
        variant="shadow"
      />
      <Input.Number
        value={line.strokeWidth ?? 1}
        onChange={handleWidthChange}
        dragScale={{ x: 0.1, y: 0.1 }}
        bounds={{ lower: 1, upper: 11 }}
        style={{ width: 140 }}
      />
      <Input.Number
        style={{ width: 100 }}
        value={line.downsample ?? 1}
        onChange={handleDownsampleChange}
        dragScale={{
          x: 0.1,
          y: 0.1,
        }}
        bounds={{
          lower: 1,
          upper: 51,
        }}
      />
    </Space>
  );
};

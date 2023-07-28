// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, ReactNode, useState } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button, Divider, Select, Space, Text, ViewportMode } from "@synnaxlabs/pluto";
import { PiSelectionPlusBold } from "react-icons/pi";
import { useDispatch } from "react-redux";

import { useSelectLineViewportMode } from "@/line/store/selectors";
import { setLineViewportMode } from "@/line/store/slice";

export const Controls = () => {
  const mode = useSelectLineViewportMode();
  const d = useDispatch();
  const [value, setValue] = useState("slope");

  const handleModeChange = (mode: ViewportMode): void => {
    d(setLineViewportMode({ mode }));
  };

  return (
    <Space style={{ paddingLeft: "2rem" }} direction="x">
      <Select.Button<
        ViewportMode,
        { key: ViewportMode; icon: ReactElement; tooltip: ReactNode }
      >
        size="medium"
        value={mode}
        onChange={handleModeChange}
        bordered={false}
        rounded={false}
        data={[
          {
            key: "zoom",
            icon: <Icon.Zoom />,
            tooltip: (
              <Space direction="x" align="center">
                <Text level="small">Zoom</Text>
                <Text.Keyboard level="small">Drag</Text.Keyboard>
              </Space>
            ),
          },
          {
            key: "pan",
            icon: <Icon.Pan />,
            tooltip: (
              <Space direction="x" align="center">
                <Text level="small">Pan</Text>
                <Text.Keyboard level="small">Shift</Text.Keyboard>
                <Text.Keyboard level="small">Drag</Text.Keyboard>
              </Space>
            ),
          },
          {
            key: "select",
            icon: <PiSelectionPlusBold />,
            tooltip: (
              <Space direction="x" align="center">
                <Text level="small">Search</Text>
                <Text.Keyboard level="small">Alt</Text.Keyboard>
                <Text.Keyboard level="small">Drag</Text.Keyboard>
              </Space>
            ),
          },
        ]}
        entryRenderKey="icon"
      >
        {({ title: _, entry, ...props }) => (
          <Button.Icon
            {...props}
            variant={props.selected ? "filled" : "text"}
            style={{}}
            size="medium"
            tooltip={entry.tooltip}
            tooltipLocation={{ x: "right", y: "top" }}
          >
            {entry.icon}
          </Button.Icon>
        )}
      </Select.Button>
      <Divider />
      <Select.Button<string, { key: string; icon: ReactElement; tooltip: ReactNode }>
        value={value}
        onChange={setValue}
        size="medium"
        bordered={false}
        rounded={false}
        entryRenderKey="icon"
        allowNone
        data={[
          {
            key: "tooltip",
            icon: <Icon.Tooltip />,
            tooltip: (
              <Space direction="x" align="center">
                <Text level="small">Tooltip</Text>
                <Text.Keyboard level="small">Alt</Text.Keyboard>
                <Text.Keyboard level="small">Click</Text.Keyboard>
              </Space>
            ),
          },
          {
            key: "slope",
            icon: <Icon.Rule />,
            tooltip: (
              <Space direction="x" align="center">
                <Text level="small">Slope</Text>
                <Text.Keyboard level="small">Alt</Text.Keyboard>
                <Text.Keyboard level="small">Drag</Text.Keyboard>
              </Space>
            ),
          },
          {
            key: "annotate",
            icon: <Icon.Annotate />,
            tooltip: (
              <Space direction="x" align="center">
                <Text level="small">Annotate</Text>
                <Text.Keyboard level="small">Alt</Text.Keyboard>
                <Text.Keyboard level="small">Click</Text.Keyboard>
              </Space>
            ),
          },
        ]}
      >
        {({ title: _, entry, ...props }) => (
          <Button.Icon
            {...props}
            key={entry.key}
            variant={props.selected ? "filled" : "text"}
            style={{}}
            size="medium"
            tooltip={entry.tooltip}
          >
            {entry.icon}
          </Button.Icon>
        )}
      </Select.Button>
      <Divider />
    </Space>
  );
};

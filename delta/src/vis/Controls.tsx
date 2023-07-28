// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ReactElement, ReactNode } from "react";

import { Icon } from "@synnaxlabs/media";
import { Button, Divider, Select, Space, Text, ViewportMode } from "@synnaxlabs/pluto";
import { PiSelectionPlusBold } from "react-icons/pi";
import { useDispatch } from "react-redux";

import { useSelectLineControlState } from "@/line/store/selectors";
import { ClickMode, setLineControlState } from "@/line/store/slice";

export const Controls = (): ReactElement => {
  const control = useSelectLineControlState();
  const d = useDispatch();

  const handleModeChange = (mode: ViewportMode): void => {
    d(setLineControlState({ state: { mode } }));
  };

  const handleClickModeChange = (clickMode: ClickMode): void => {
    d(setLineControlState({ state: { clickMode } }));
  };

  const handleTooltipChange = (tooltip: boolean): void => {
    d(setLineControlState({ state: { enableTooltip: tooltip } }));
  };

  return (
    <Space style={{ paddingLeft: "2rem" }} direction="x" size="small">
      <Select.Button<
        ViewportMode,
        { key: ViewportMode; icon: ReactElement; tooltip: ReactNode }
      >
        size="medium"
        value={control.mode}
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
            key={entry.key}
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
      <Button.ToggleIcon
        value={control.enableTooltip}
        onChange={handleTooltipChange}
        checkedVariant="filled"
        uncheckedVariant="text"
        sharp
        tooltip={
          <Space direction="x" align="center">
            <Text level="small">Tooltip</Text>
            <Text.Keyboard level="small">Alt</Text.Keyboard>
            <Text.Keyboard level="small">Click</Text.Keyboard>
          </Space>
        }
      >
        <Icon.Tooltip />
      </Button.ToggleIcon>
      <Divider />
      <Select.Button<
        ClickMode,
        { key: ClickMode; icon: ReactElement; tooltip: ReactNode }
      >
        value={control.clickMode as ClickMode}
        onChange={handleClickModeChange}
        size="medium"
        bordered={false}
        rounded={false}
        entryRenderKey="icon"
        allowNone
        data={[
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
            tooltipLocation={{ x: "right", y: "top" }}
          >
            {entry.icon}
          </Button.Icon>
        )}
      </Select.Button>
    </Space>
  );
};

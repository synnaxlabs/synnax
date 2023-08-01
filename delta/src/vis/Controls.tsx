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
import {
  Button,
  Divider,
  Select,
  Space,
  Text,
  Typography,
  ViewportMode,
} from "@synnaxlabs/pluto";
import { XYLocation } from "@synnaxlabs/x";
import { PiSelectionPlusBold } from "react-icons/pi";
import { useDispatch } from "react-redux";

import { useSelectActiveMosaicTabKey } from "@/layout";
import { useSelectLineControlState } from "@/line/store/selectors";
import {
  ClickMode,
  setLineControlState,
  setLinePlotViewport,
} from "@/line/store/slice";

export const Controls = (): ReactElement => {
  const control = useSelectLineControlState();
  const vis = useSelectActiveMosaicTabKey();
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

  const handleZoomReset = (): void => {
    if (vis != null) d(setLinePlotViewport({ layoutKey: vis }));
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
            size="medium"
            tooltip={entry.tooltip}
            tooltipLocation={{ x: "right", y: "top" }}
          >
            {entry.icon}
          </Button.Icon>
        )}
      </Select.Button>
      <Button.Icon
        onClick={handleZoomReset}
        variant="text"
        tooltipLocation={{ x: "right", y: "top" }}
        tooltip={
          <Space direction="x" align="center">
            <Text level="small">Reset Zoom</Text>
            <Text.Keyboard level="small">
              <Typography.Symbols.Meta />
            </Text.Keyboard>
            <Text.Keyboard level="small">Click</Text.Keyboard>
          </Space>
        }
        size="medium"
      >
        <Icon.ZoomReset />
      </Button.Icon>

      <Divider />
      <Button.ToggleIcon
        value={control.enableTooltip}
        onChange={handleTooltipChange}
        checkedVariant="filled"
        uncheckedVariant="text"
        sharp
        tooltip={
          <Space direction="x" align="center">
            <Text level="small">Show Tooltip on Hover</Text>
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
            key: "measure",
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

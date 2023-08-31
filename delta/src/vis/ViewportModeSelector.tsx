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
import { Align, Button, Input, Select, Text, Viewport } from "@synnaxlabs/pluto";
import { PiSelectionPlusBold } from "react-icons/pi";
import { useDispatch } from "react-redux";

import { useSelectViewportMode } from "./selectors";
import { setViewportMode } from "./slice";

export const ViewportModeSelector = (): ReactElement => {
  const d = useDispatch();
  const m = useSelectViewportMode();

  const handleChange = (mode: Viewport.Mode): void => {
    d(setViewportMode({ mode }));
  };

  return (
    <Select.Button<
      Viewport.Mode,
      { key: Viewport.Mode; icon: ReactElement; tooltip: ReactNode }
    >
      size="medium"
      value={m}
      onChange={handleChange}
      bordered={false}
      rounded={false}
      data={[
        {
          key: "zoom",
          icon: <Icon.Zoom />,
          tooltip: (
            <Align.Space direction="x" align="center">
              <Text.Text level="small">Zoom</Text.Text>
              <Text.Keyboard level="small">Drag</Text.Keyboard>
            </Align.Space>
          ),
        },
        {
          key: "pan",
          icon: <Icon.Pan />,
          tooltip: (
            <Align.Space direction="x" align="center">
              <Text.Text level="small">Pan</Text.Text>
              <Text.Keyboard level="small">Shift</Text.Keyboard>
              <Text.Keyboard level="small">Drag</Text.Keyboard>
            </Align.Space>
          ),
        },
        {
          key: "select",
          icon: <PiSelectionPlusBold />,
          tooltip: (
            <Align.Space direction="x" align="center">
              <Text.Text level="small">Search</Text.Text>
              <Text.Keyboard level="small">Alt</Text.Keyboard>
              <Text.Keyboard level="small">Drag</Text.Keyboard>
            </Align.Space>
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
  );
};

// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/hardware/ni/task/DigitalListItem.css";

import { Align, Form, List, Text } from "@synnaxlabs/pluto";
import { type ReactElement, type ReactNode } from "react";

import { CSS } from "@/css";
import { Common } from "@/hardware/common";
import { type DIChannel, type DOChannel } from "@/hardware/ni/task/types";

export interface DigitalListItemProps
  extends Common.Task.ChannelListItemProps<DIChannel | DOChannel> {
  children: ReactNode;
}

export const DigitalListItem = ({
  path,
  children,
  isSnapshot,
  ...props
}: DigitalListItemProps): ReactElement => {
  const { enabled } = props.entry;
  const { set } = Form.useContext();
  return (
    <List.ItemFrame
      {...props}
      style={{ width: "100%" }}
      justify="spaceBetween"
      align="center"
      direction="x"
    >
      <Align.Space direction="x" align="center" justify="spaceEvenly">
        <Align.Pack
          className="port-line-input"
          direction="x"
          align="center"
          style={{ maxWidth: "50rem" }}
        >
          <Form.NumericField
            path={`${path}.port`}
            showLabel={false}
            showHelpText={false}
            inputProps={{ showDragHandle: false }}
            hideIfNull
          />
          <Text.Text level="p">/</Text.Text>
          <Form.NumericField
            path={`${path}.line`}
            showHelpText={false}
            showLabel={false}
            inputProps={{ showDragHandle: false }}
            hideIfNull
          />
        </Align.Pack>
        <Text.Text
          level="small"
          className={CSS.BE("port-line-input", "label")}
          shade={7}
          weight={450}
        >
          Port/Line
        </Text.Text>
      </Align.Space>
      <Align.Space direction="x" align="center" justify="spaceEvenly">
        {children}
        <Common.Task.EnableDisableButton
          value={enabled}
          onChange={(v) => set(`${path}.enabled`, v)}
          isSnapshot={isSnapshot}
        />
      </Align.Space>
    </List.ItemFrame>
  );
};

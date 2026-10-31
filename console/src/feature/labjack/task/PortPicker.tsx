// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/feature/labjack/task/PortPicker.css";

import { Dialog } from "@synnaxlabs/lyra/dialog";
import { Flex } from "@synnaxlabs/lyra/flex";
import { Form } from "@synnaxlabs/lyra/form";
import { Input } from "@synnaxlabs/lyra/input";
import { type ReactElement, type ReactNode } from "react";

import {
  type Model,
  type Port,
  PORTS,
  type PortType,
} from "@/feature/labjack/device/types";
import { PortBanks } from "@/feature/labjack/task/PortBanks";
import { claims, describer, terminal } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";
import { CSS } from "@/platform/css";

interface GridProps<C extends Channel> {
  model: Model;
  ports: Port[];
  channel: string;
  name: (channel: C) => string;
  onChange: (port: string) => void;
}

const Grid = <C extends Channel>({
  model,
  ports,
  channel,
  name,
  onChange,
}: GridProps<C>): ReactElement => {
  const { close } = Dialog.useContext();
  const channels = Form.useFieldValue<C[]>("config.channels");
  return (
    <PortBanks
      ports={ports}
      claims={claims(channels, model)}
      selected={[channel]}
      describe={describer(channels, name)}
      onClick={(cell) => {
        onChange(cell.key);
        close();
      }}
    />
  );
};

export interface PortPickerProps<C extends Channel> {
  value: string;
  onChange: (port: string) => void;
  preview?: boolean;
  model: Model;
  type: PortType;
  /** The key of the channel whose port it is. */
  channel: string;
  /** The name a channel shows by. */
  name: (channel: C) => string;
  /** Packed after the trigger, such as the channel's type buttons. */
  children?: ReactNode;
  className?: string;
}

/**
 * The port of a channel as a button that names its terminal. The button opens the grid
 * of the device's ports of the channel's type, and a click on a port picks it.
 */
export const PortPicker = <C extends Channel>({
  value,
  onChange,
  preview,
  model,
  type,
  channel,
  name,
  children,
  className,
}: PortPickerProps<C>): ReactElement => {
  const ports = PORTS[model][type];
  const port = ports.find(({ key }) => key === value);
  const itemStatus = Input.useItemStatus({});
  return (
    <Dialog.Frame className={className}>
      <Flex.Box pack x>
        <Dialog.Trigger preview={preview} {...itemStatus}>
          {port == null ? value : terminal(port)}
        </Dialog.Trigger>
        {children}
      </Flex.Box>
      <Dialog.Dialog className={CSS.BE("labjack-port-picker", "dialog")}>
        <Grid<C>
          model={model}
          ports={ports}
          channel={channel}
          name={name}
          onChange={onChange}
        />
      </Dialog.Dialog>
    </Dialog.Frame>
  );
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Model, PORTS, type PortType } from "@/feature/labjack/device/types";
import { BankTabs, PortBanks } from "@/feature/labjack/task/PortBanks";
import { claims } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";
import { Task } from "@/platform/task";

export interface PortFieldProps<C extends Channel> {
  /** The channel's path in the form. */
  path: deep.Target<C>;
  /** The path of the task's channels. */
  channelsPath: deep.Target<C[]>;
  model: Model;
  type: PortType;
  /** The name a channel shows by. */
  name: (channel: C) => string;
}

/**
 * The port of a channel, picked from the grid of the device's ports of its type. When
 * the ports include an accessory's, such as a Mux80's, the grid shows one bank at a
 * time.
 */
export const PortField = <C extends Channel>({
  path,
  channelsPath,
  model,
  type,
  name,
}: PortFieldProps<C>): ReactElement => {
  const { key: keyPath, port } = deep.widen<Channel>(path);
  const key = Form.useFieldValue(keyPath);
  const channels = Form.useFieldValue(channelsPath);
  const ports = PORTS[model][type];
  const tabbed = ports.some(({ connector }) => connector != null);
  return (
    <Form.Field path={port} label="Port">
      {({ value, onChange, preview }) => {
        const props = {
          ports,
          claims: claims(channels, model),
          selected: [key],
          describe: Task.Port.describer(channels, name),
          onMove: preview ? undefined : (cell: Task.Port.Cell) => onChange(cell.key),
        };
        return tabbed ? (
          <BankTabs key={key} port={value} {...props} />
        ) : (
          <PortBanks {...props} />
        );
      }}
    </Form.Field>
  );
};

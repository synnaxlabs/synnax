// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type Device } from "@/feature/ni/device";
import { claimsOf, groups, type PortKind } from "@/feature/ni/task/ports";
import { Task } from "@/platform/task";

export interface PortFieldProps<C extends Task.Channel> {
  /** The channel's path in the form. */
  path: deep.Target<{ key: string; port: number }>;
  /** The path of the task's channels. */
  channelsPath: deep.Target<C[]>;
  device?: Device.Device;
  kind: PortKind<C>;
  deviceOf: (channel: C) => device.Key;
}

/**
 * The port of a channel, picked from the grid of its device's ports. A device with no
 * port list takes a number instead.
 */
export const PortField = <C extends Task.Channel>({
  path,
  channelsPath,
  device,
  kind,
  deviceOf,
}: PortFieldProps<C>): ReactElement => {
  const { key: keyPath, port } = deep.widen(path);
  const key = Form.useFieldValue(keyPath);
  const channels = Form.useFieldValue(channelsPath);
  const hardware = device?.properties.hardware;
  if (device == null || hardware == null)
    return <Form.NumericField path={port} label="Port" />;
  const own = channels.filter((c) => deviceOf(c) === device.key);
  const portGroups = groups({
    ports: kind.ports(hardware),
    claims: claimsOf(kind, own, hardware),
    prefix: kind.prefix,
    selected: [key],
    describe: Task.Port.describer(own, (c) => kind.name(c, device)),
  });
  return (
    <Form.Field path={port} label="Port">
      {({ onChange, preview }) => (
        <Task.Port.Grid
          label="Port"
          groups={portGroups}
          onClick={(cell) => {
            if (!preview) onChange(Number(cell.key));
          }}
        />
      )}
    </Form.Field>
  );
};

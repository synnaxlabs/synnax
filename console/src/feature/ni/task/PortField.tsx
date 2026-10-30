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
import { type ReactElement } from "react";

import { type Device } from "@/feature/ni/device/types";
import { describer, groups, type PortKind } from "@/feature/ni/task/ports";
import { Task } from "@/platform/task";

export interface PortFieldProps<C extends Task.Channel> {
  /** The channel's path in the form. */
  path: string;
  device?: Device;
  kind: PortKind<C>;
  deviceOf: (channel: C) => device.Key;
}

/**
 * The port of a channel, picked from the grid of its device's ports. A device with no
 * port list takes a number instead.
 */
export const PortField = <C extends Task.Channel>({
  path,
  device,
  kind,
  deviceOf,
}: PortFieldProps<C>): ReactElement => {
  const key = Form.useFieldValue<string>(`${path}.key`);
  const channels = Form.useFieldValue<C[]>("config.channels");
  const hardware = device?.properties.hardware;
  if (device == null || hardware == null)
    return <Form.NumericField path={`${path}.port`} label="Port" />;
  const own = channels.filter((c) => deviceOf(c) === device.key);
  const claims = kind.claims(own, device.key, hardware);
  const portGroups = groups({
    ports: kind.ports(hardware),
    claims,
    prefix: kind.prefix,
    selected: [key],
    describe: describer(kind, own, device),
  });
  return (
    <Form.Field<number> path={`${path}.port`} label="Port">
      {({ onChange, preview }) => (
        <Task.PortGrid
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

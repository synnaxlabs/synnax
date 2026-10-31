// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type ReactElement } from "react";

import { type Model, PORTS, type PortType } from "@/feature/labjack/device/types";
import { PortBanks } from "@/feature/labjack/task/PortBanks";
import { claims, describer } from "@/feature/labjack/task/ports";
import { type Channel } from "@/feature/labjack/task/types";

export interface PortFieldProps<C extends Channel> {
  /** The channel's path in the form. */
  path: string;
  model: Model;
  type: PortType;
  /** The name a channel shows by. */
  name: (channel: C) => string;
}

/** The port of a channel, picked from the grid of the device's ports of its type. */
export const PortField = <C extends Channel>({
  path,
  model,
  type,
  name,
}: PortFieldProps<C>): ReactElement => {
  const key = Form.useFieldValue<string>(`${path}.key`);
  const channels = Form.useFieldValue<C[]>("config.channels");
  return (
    <Form.Field<string> path={`${path}.port`} label="Port">
      {({ onChange, preview }) => (
        <PortBanks
          ports={PORTS[model][type]}
          claims={claims(channels, model)}
          selected={[key]}
          describe={describer(channels, name)}
          onClick={(cell) => {
            if (!preview) onChange(cell.key);
          }}
        />
      )}
    </Form.Field>
  );
};

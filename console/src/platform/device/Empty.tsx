// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Form } from "@synnaxlabs/lyra/form";
import { Access, Device } from "@synnaxlabs/pluto";
import { deep } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { type SelectConnect } from "@/platform/device/Select";
import { useHandleChange } from "@/platform/device/useHandleChange";
import { Empty } from "@/platform/empty";
import { Task } from "@/platform/task";

const path = deep.path<{ config: { device: device.Key } }>();

/** What a task form offers while it has no device. */
export interface NoneSelectedParams {
  /** The make of the devices the task can use. */
  make: string;
  /** The message when no device of the make exists. */
  emptyContent: string;
  connect: SelectConnect;
}

export interface NoneSelectedProps extends NoneSelectedParams {
  onConfigure: (key: device.Key) => void;
}

/**
 * The body of a task form that has no device. With no device of the make, it offers to
 * connect one and selects it. Otherwise, it opens the device select at
 * `config.device`. A read-only form offers neither.
 */
export const NoneSelected = ({
  make,
  emptyContent,
  connect,
  onConfigure,
}: NoneSelectedProps): ReactElement => {
  const ctx = Form.useContext();
  const isPreview = Task.useIsPreview(ctx);
  const canConnect = Access.useCreateGranted(device.TYPE_RESOURCE_ID);
  const handleChange = useHandleChange(onConfigure);
  const devices = Device.useMultiple({ where: { make: { eq: make } }, limit: 1 });
  if (devices.length > 0)
    return (
      <Empty.Action
        message="No device selected"
        action={isPreview ? "" : "Select device"}
        onClick={() => ctx.focus(path.config.device)}
      />
    );
  const handleConnect = async () => {
    const key = await connect.open();
    if (key == null) return;
    ctx.set(path.config.device, key);
    handleChange(key, ctx);
  };
  return (
    <Empty.Action
      message={emptyContent}
      action={canConnect && !isPreview ? `Connect ${connect.title}` : ""}
      onClick={() => void handleConnect()}
    />
  );
};

export interface UnconfiguredProps {
  device: Pick<device.Device, "key" | "name">;
  canConfigure: boolean;
  onConfigure: (key: device.Key) => void;
}

export const Unconfigured = ({
  device: { key, name },
  canConfigure,
  onConfigure,
}: UnconfiguredProps): ReactElement => (
  <Empty.Action
    message={`${name} is not configured`}
    action={canConfigure ? `Configure ${name}` : ""}
    onClick={() => onConfigure(key)}
  />
);

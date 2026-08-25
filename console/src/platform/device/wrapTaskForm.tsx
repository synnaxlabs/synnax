// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { device } from "@synnaxlabs/client";
import { Access, Form } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { type FC } from "react";

import { NoneSelected, Unconfigured } from "@/platform/device/Empty";
import { Task } from "@/platform/task";

export interface TaskFormContentProps<D> {
  device: D;
}

export interface WrapTaskFormParams<D> {
  use: (query: { key: device.Key }) => D;
  useConfigure: () => (params: { deviceKey: device.Key }) => void;
  Content: FC<TaskFormContentProps<D>>;
}

/**
 * Wraps a task form body that can only render against a configured device, taking the
 * device key from the form's config.device field. Until that device resolves and
 * reports itself configured, the wrapper renders a prompt in place of the body.
 */
export const wrapTaskForm = <
  D extends Pick<device.Device, "key" | "name" | "configured">,
>({
  use,
  useConfigure,
  Content,
}: WrapTaskFormParams<D>): FC<{}> => {
  const Configured = ({ deviceKey }: { deviceKey: device.Key }) => {
    const isSnapshot = Task.useIsSnapshot();
    const canUpdate = Access.useUpdateGranted(device.ontologyID(deviceKey));
    const configure = useConfigure();
    const dev = use({ key: deviceKey });
    if (!dev.configured)
      return (
        <Unconfigured
          device={dev}
          canConfigure={!isSnapshot && canUpdate}
          onConfigure={(deviceKey) => configure({ deviceKey })}
        />
      );
    return <Content device={dev} />;
  };
  Configured.displayName = `Configured(${Content.displayName ?? Content.name})`;
  const Wrapper = () => {
    const deviceKey = Form.useFieldValue<string>("config.device");
    if (primitive.isZero(deviceKey)) return <NoneSelected />;
    return <Configured deviceKey={deviceKey} />;
  };
  Wrapper.displayName = `DeviceTaskForm(${Content.displayName ?? Content.name})`;
  return Wrapper;
};

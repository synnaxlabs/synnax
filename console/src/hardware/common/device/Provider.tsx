// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Align, Form, Text } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { Device } from "@/hardware/common/device";
import { Layout } from "@/layout";

export interface ProviderProps<
  P extends UnknownRecord = UnknownRecord,
  M extends string = string,
> {
  configureLayout: Omit<Layout.BaseState, "key">;
  snapshot?: boolean;
  noDeviceSelectedMessage?: string;
  children: (props: { device: device.Device<P, M> }) => ReactElement;
}

export const Provider = <
  P extends UnknownRecord = UnknownRecord,
  M extends string = string,
>({
  configureLayout,
  snapshot,
  children,
  noDeviceSelectedMessage = "No device selected",
}: ProviderProps<P, M>): ReactElement => {
  const formCtx = Form.useContext();
  const device = Device.use<P, M>(formCtx);
  const placeLayout = Layout.usePlacer();
  if (device == null)
    return (
      <Align.Space grow empty align="center" justify="center">
        <Text.Text level="p">{noDeviceSelectedMessage}</Text.Text>
      </Align.Space>
    );
  const handleConfigure = () => placeLayout({ ...configureLayout, key: device.key });
  if (!device.configured)
    return (
      <Align.Space grow align="center" justify="center" direction="y">
        <Text.Text level="p">{`${device.name} is not configured.`}</Text.Text>
        {snapshot !== true && (
          <Text.Link level="p" onClick={handleConfigure}>
            {`Configure ${device.name}.`}
          </Text.Link>
        )}
      </Align.Space>
    );
  return children({ device });
};

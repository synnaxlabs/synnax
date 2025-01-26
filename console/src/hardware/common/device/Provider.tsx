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

import { use, type UseContextValue } from "@/hardware/common/device/use";
import { Layout } from "@/layout";

export interface ProviderProps<
  P extends UnknownRecord = UnknownRecord,
  MK extends string = string,
  MO extends string = string,
> {
  configureLayout: Omit<Layout.BaseState, "key">;
  isSnapshot: boolean;
  noneSelectedElement?: ReactElement;
  children: (props: { device: device.Device<P, MK, MO> }) => ReactElement;
}

const DEFAULT_NONE_SELECTED_ELEMENT = (
  <Align.Space grow empty align="center" justify="center">
    <Text.Text level="p">No device selected.</Text.Text>
  </Align.Space>
);

export const Provider = <
  P extends UnknownRecord = UnknownRecord,
  MK extends string = string,
  MO extends string = string,
>({
  configureLayout,
  isSnapshot: snapshot,
  children,
  noneSelectedElement = DEFAULT_NONE_SELECTED_ELEMENT,
}: ProviderProps<P, MK, MO>): ReactElement => {
  const formCtx = Form.useContext<UseContextValue>();
  const device = use<P, MK, MO>(formCtx);
  const placeLayout = Layout.usePlacer();
  if (device == null) return noneSelectedElement;
  const handleConfigure = () => placeLayout({ ...configureLayout, key: device.key });
  if (!device.configured)
    return (
      <Align.Space grow align="center" justify="center" direction="y">
        <Text.Text level="p">{`${device.name} is not configured.`}</Text.Text>
        {!snapshot && (
          <Text.Link level="p" onClick={handleConfigure}>
            {`Configure ${device.name}.`}
          </Text.Link>
        )}
      </Align.Space>
    );
  return children({ device });
};

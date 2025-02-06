// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Align, Text } from "@synnaxlabs/pluto";
import { type UnknownRecord } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { use } from "@/hardware/common/device/use";
import { Layout } from "@/layout";

const DEFAULT_NONE_SELECTED_ELEMENT = (
  <Align.Center>
    <Text.Text level="p">No device selected.</Text.Text>
  </Align.Center>
);

export interface ProviderChildProps<
  P extends UnknownRecord = UnknownRecord,
  MK extends string = string,
  MO extends string = string,
> {
  device: device.Device<P, MK, MO>;
}

export interface ProviderProps<
  P extends UnknownRecord = UnknownRecord,
  MK extends string = string,
  MO extends string = string,
> {
  canConfigure: boolean;
  children: (props: ProviderChildProps<P, MK, MO>) => ReactElement;
  configureLayout: Omit<Layout.BaseState, "key">;
  noneSelectedElement?: ReactElement;
}

export const Provider = <
  P extends UnknownRecord = UnknownRecord,
  MK extends string = string,
  MO extends string = string,
>({
  canConfigure,
  children,
  configureLayout,
  noneSelectedElement = DEFAULT_NONE_SELECTED_ELEMENT,
}: ProviderProps<P, MK, MO>): ReactElement => {
  const device = use<P, MK, MO>();
  const placeLayout = Layout.usePlacer();
  if (device == null) return noneSelectedElement;
  if (!device.configured) {
    const { name } = device;
    const handleConfigure = () => placeLayout({ ...configureLayout, key: device.key });
    return (
      <Align.Center>
        <Text.Text level="p">{`${name} is not configured.`}</Text.Text>
        {canConfigure && (
          <Text.Link level="p" onClick={handleConfigure}>
            {`Configure ${name}.`}
          </Text.Link>
        )}
      </Align.Center>
    );
  }
  return children({ device });
};

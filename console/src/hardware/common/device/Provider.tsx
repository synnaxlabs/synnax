// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/pluto";
import { type record } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { EmptyAction } from "@/components";
import { use } from "@/hardware/common/device/use";
import { Layout } from "@/layout";

const DEFAULT_NONE_SELECTED_CONTENT = (
  <Text.Text center color={8}>
    No device selected.
  </Text.Text>
);

export interface ProviderChildProps<
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
> {
  device: device.Device<Properties, Make, Model>;
}

export interface ProviderProps<
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
> {
  canConfigure: boolean;
  children: (props: ProviderChildProps<Properties, Make, Model>) => ReactElement;
  configureLayout: Layout.BaseState;
  noneSelectedContent?: ReactElement;
}

export const Provider = <
  Properties extends record.Unknown = record.Unknown,
  Make extends string = string,
  Model extends string = string,
>({
  canConfigure,
  children,
  configureLayout,
  noneSelectedContent = DEFAULT_NONE_SELECTED_CONTENT,
}: ProviderProps<Properties, Make, Model>) => {
  const device = use<Properties, Make, Model>();
  const placeLayout = Layout.usePlacer();
  if (device == null) return noneSelectedContent;
  if (!device.configured) {
    const { name } = device;
    const handleConfigure = () => placeLayout({ ...configureLayout, key: device.key });
    return (
      <EmptyAction
        message={`${name} is not configured.`}
        action={canConfigure ? `Configure ${name}` : ""}
        onClick={handleConfigure}
      />
    );
  }
  return children({ device });
};

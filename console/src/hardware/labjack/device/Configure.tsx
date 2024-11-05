// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Configure as Core } from "@/hardware/device/Configure";
import { type Properties, ZERO_PROPERTIES } from "@/hardware/labjack/device/types";
import { type Layout } from "@/layout";

export const Configure = ({
  layoutKey,
  onClose,
}: Layout.RendererProps): ReactElement => {
  const client = Synnax.use();
  const { data: device, isPending } = useQuery({
    queryKey: [layoutKey, client?.key],
    queryFn: async () => {
      if (client == null) return;
      return await client.hardware.devices.retrieve<Properties>(layoutKey);
    },
  });
  if (isPending || device == null) return <div>Loading...</div>;
  return <Core device={device} onClose={onClose} zeroProperties={ZERO_PROPERTIES} />;
};

export const CONFIGURE_LAYOUT_TYPE = "configure_LabJack";
export type LayoutType = typeof CONFIGURE_LAYOUT_TYPE;

export const createConfigureLayout =
  (key: string, initial: Omit<Partial<Layout.State>, "type">) => (): Layout.State => {
    const { name = "LabJack.Device.Configure", location = "modal", ...rest } = initial;
    return {
      key,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: key, //TODO: difference between key and windowKey?
      name,
      icon: "Logo.LabJack",
      window: {
        navTop: true,
        size: { height: 350, width: 800 },
        resizable: true,
      },
      location,
      ...rest,
    };
  };

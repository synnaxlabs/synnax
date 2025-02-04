// Copyright 2025 Synnax Labs, Inc.
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
import { type Properties, ZERO_PROPERTIES } from "@/hardware/ni/device/types";
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

export const CONFIGURE_LAYOUT_TYPE = "configure_NI";
export type LayoutType = typeof CONFIGURE_LAYOUT_TYPE;

export const createConfigureLayout =
  (device: string, initial: Omit<Partial<Layout.State>, "type">) =>
  (): Layout.State => {
    const { name = "Configure Device", location = "modal", ...rest } = initial;
    return {
      key: initial.key ?? device,
      type: CONFIGURE_LAYOUT_TYPE,
      windowKey: initial.key ?? device,
      name,
      icon: "Logo.NI",
      window: {
        navTop: true,
        size: { height: 350, width: 800 },
        resizable: true,
      },
      location,
      ...rest,
    };
  };

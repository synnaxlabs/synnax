// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Device, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import {
  type PropsWithChildren,
  type ReactElement,
  useCallback,
  useState,
} from "react";

import { Cluster } from "@/cluster";
import { Toolbar } from "@/components";
import { StateContext, type States } from "@/hardware/device/StateContext";
import { Rack } from "@/hardware/rack";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

const StateProvider = (props: PropsWithChildren) => {
  const client = Synnax.use();
  const [states, setStates] = useState<States>({});
  useAsyncEffect(async () => {
    if (client == null) return;
    const devices = await client.hardware.devices.retrieve([], { includeState: true });
    const initialStates: States = Object.fromEntries(
      devices
        .filter(({ state }) => state != null)
        .map(({ key, state }) => [key, state as device.State]),
    );
    setStates(initialStates);
  }, [client]);
  const handleStateUpdate = useCallback((state: device.State) => {
    setStates((prevStates) => ({ ...prevStates, [state.key]: state }));
  }, []);
  Device.useStateSynchronizer(handleStateUpdate);
  return <StateContext {...props} value={states} />;
};

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery<ontology.ID | null>({
    queryKey: [client?.key, "device-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client?.ontology.retrieveChildren(ontology.ROOT_ID, {
        includeSchema: false,
      });
      return res?.find(({ name }) => name === "Devices")?.id ?? null;
    },
  });

  return (
    <Cluster.NoneConnectedBoundary>
      <StateProvider>
        <Rack.StateProvider>
          <Align.Space empty style={{ height: "100%" }}>
            <Toolbar.Header>
              <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
            </Toolbar.Header>
            <Ontology.Tree root={group.data ?? undefined} />
          </Align.Space>
        </Rack.StateProvider>
      </StateProvider>
    </Cluster.NoneConnectedBoundary>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "device",
  icon: <Icon.Device />,
  content: <Content />,
  tooltip: "Devices",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
  trigger: ["D"],
};

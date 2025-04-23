// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type device, ontology, type rack } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Device, Rack, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import {
  createContext,
  type PropsWithChildren,
  type ReactElement,
  use,
  useCallback,
  useState as reactUseState,
} from "react";

import { Cluster } from "@/cluster";
import { Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

export interface DeviceStates extends Record<device.Key, device.State> {}

const StateContext = createContext<DeviceStates>({});

const StateProvider = (props: PropsWithChildren) => {
  const client = Synnax.use();
  const [states, setStates] = reactUseState<DeviceStates>({});
  useAsyncEffect(async () => {
    if (client == null) return;
    const devices = await client.hardware.devices.retrieve([], { includeState: true });
    const initialStates: DeviceStates = Object.fromEntries(
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

interface RackStates extends Record<string, rack.State> {}

const RackStateContext = createContext<RackStates>({});

const RackStateProvider = (props: PropsWithChildren) => {
  const client = Synnax.use();
  const [states, setStates] = reactUseState<RackStates>({});
  useAsyncEffect(async () => {
    if (client == null) return;
    const racks = await client.hardware.racks.retrieve([], { includeState: true });
    const initialStates: RackStates = Object.fromEntries(
      racks
        .filter(({ state }) => state != null)
        .map(({ key, state }) => [key, state as rack.State]),
    );
    setStates(initialStates);
  }, [client]);

  const handleStateUpdate = useCallback((state: rack.State) => {
    setStates((prevStates) => ({ ...prevStates, [state.key]: state }));
  }, []);
  Rack.useStateSynchronizer(handleStateUpdate);

  return <RackStateContext {...props} value={states} />;
};

export const useRackState = (key: string): rack.State | undefined =>
  use(RackStateContext)[key];

export const useState = (key: device.Key): device.State | undefined =>
  use(StateContext)[key];

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery<ontology.ID | undefined>({
    queryKey: [client?.key, "device-group"],
    queryFn: async () => {
      if (client == null) return undefined;
      const res = await client?.ontology.retrieveChildren(ontology.ROOT_ID, {
        includeSchema: false,
      });
      return res?.find(({ name }) => name === "Devices")?.id;
    },
  });

  return (
    <Cluster.NoneConnectedBoundary>
      <StateProvider>
        <RackStateProvider>
          <Align.Space empty style={{ height: "100%" }}>
            <Toolbar.Header>
              <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
            </Toolbar.Header>
            <Ontology.Tree root={group.data} />
          </Align.Space>
        </RackStateProvider>
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

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
import { Align, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import {
  createContext,
  type ReactElement,
  useContext,
  useState as reactUseState,
} from "react";

import { Cluster } from "@/cluster";
import { Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

export interface StateContextValue extends Record<string, device.Status> {}

const StateContext = createContext<StateContextValue>({});

const StateProvider = ({ children }: { children: ReactElement }) => {
  const client = Synnax.use();
  const [states, setStates] = reactUseState<StateContextValue>({});

  useAsyncEffect(async () => {
    if (client == null) return;
    const devs = await client.hardware.devices.retrieve([], { includeStatus: true });
    const initialStates: StateContextValue = Object.fromEntries(
      devs
        .filter((d) => d.status != null)
        .map((d) => [d.key, d.status as device.Status]),
    );
    setStates(initialStates);
    const observer = await client.hardware.devices.openStateObserver();
    const disconnect = observer.onChange((states) => {
      setStates((prevStates) => {
        const nextStates = Object.fromEntries(states.map((s) => [s.key, s]));
        return { ...prevStates, ...nextStates };
      });
    });
    return async () => {
      disconnect();
      await observer.close();
    };
  }, []);

  return <StateContext.Provider value={states}>{children}</StateContext.Provider>;
};

interface RackStateContextValue extends Record<string, rack.Status> {}

const RackStateContext = createContext<RackStateContextValue>({});

const RackStateProvider = ({ children }: { children: ReactElement }) => {
  const client = Synnax.use();
  const [states, setStates] = reactUseState<RackStateContextValue>({});

  useAsyncEffect(async () => {
    if (client == null) return;
    const racks = await client.hardware.racks.retrieve([], { includeStatus: true });
    const initialStates: RackStateContextValue = Object.fromEntries(
      racks
        .filter((r) => r.status != null)
        .map((r) => [r.key, r.status as rack.Status]),
    );
    setStates(initialStates);
    const observer = await client.hardware.racks.openStateObserver();
    const disconnect = observer.onChange((states) => {
      setStates((prevStates) => {
        const nextStates = Object.fromEntries(states.map((s) => [s.key, s]));
        return { ...prevStates, ...nextStates };
      });
    });
    return async () => {
      disconnect();
      await observer.close();
    };
  }, []);

  return (
    <RackStateContext.Provider value={states}>{children}</RackStateContext.Provider>
  );
};

export const useRackState = (key: string): rack.Status | undefined =>
  useContext(RackStateContext)[key];

export const useState = (key: string): device.Status | undefined =>
  useContext(StateContext)[key];

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery({
    queryKey: [client?.key, "device-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client.ontology.retrieveChildren(ontology.ROOT_ID);
      return res.filter((r) => r.name === "Devices")[0].id;
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

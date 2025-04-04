import { type device, ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Header, Synnax, useAsyncEffect } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import {
  createContext,
  type ReactElement,
  useContext,
  useState as reactUseState,
} from "react";

import { Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

export interface StateProviderContextValue {
  states: Record<string, device.State>;
}

const StateContext = createContext<StateProviderContextValue>({
  states: {},
});

const StateProvider = ({ children }: { children: ReactElement }) => {
  const client = Synnax.use();
  const [states, setStates] = reactUseState<Record<string, device.State>>({});

  useAsyncEffect(async () => {
    if (client == null) return;
    const devs = await client?.hardware.devices.retrieve([], {
      includeState: true,
    });
    const initialStates = Object.fromEntries(
      devs.filter((d) => d.state != null).map((d) => [d.key, d.state]),
    ) as Record<string, device.State>;
    setStates(initialStates);
    const observer = await client.hardware.devices.openStateObserver();
    const disconnect = observer.onChange((state) => {
      setStates((prev) => {
        const newStates = Object.fromEntries(state.map((s) => [s.key, s]));
        return { ...prev, ...newStates };
      });
    });
    return async () => {
      disconnect();
      await observer.close();
    };
  }, []);

  return <StateContext.Provider value={{ states }}>{children}</StateContext.Provider>;
};
interface RackHeartbeatProviderContextValue {
  heartbeats: Record<number, number>;
}

const RackHeartbeatContext = createContext<RackHeartbeatProviderContextValue>({
  heartbeats: {},
});

const RackHeartbeatProvider = ({ children }: { children: ReactElement }) => {
  const client = Synnax.use();
  const [heartbeats, setHeartbeats] = reactUseState<Record<number, number>>({});

  useAsyncEffect(async () => {
    if (client == null) return;
    const observer = await client.hardware.racks.openHeartbeatObserver();
    const disconnect = observer.onChange((beats) => {
      setHeartbeats((prev) => {
        const newBeats = Object.fromEntries(beats.map((b) => [b.rackKey, b.heartbeat]));
        return { ...prev, ...newBeats };
      });
    });
    return async () => {
      disconnect();
      await observer.close();
    };
  }, []);

  return (
    <RackHeartbeatContext.Provider value={{ heartbeats }}>
      {children}
    </RackHeartbeatContext.Provider>
  );
};

export const useHeartbeat = (key: number): number | undefined => {
  const { heartbeats } = useContext(RackHeartbeatContext);
  return heartbeats[key];
};

export const useState = (key: string): device.State | undefined => {
  const { states } = useContext(StateContext);
  return states[key];
};

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery<ontology.ID | undefined>({
    queryKey: [client?.key, "device-group"],
    queryFn: async () => {
      if (client == null) return undefined;
      const res = await client?.ontology.retrieveChildren(ontology.ROOT_ID, {
        includeSchema: false,
      });
      return res?.filter((r) => r.name === "Devices")[0].id;
    },
  });

  return (
    <StateProvider>
      <RackHeartbeatProvider>
        <Align.Space empty style={{ height: "100%" }}>
          <Toolbar.Header>
            <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
            <Header.Actions></Header.Actions>
          </Toolbar.Header>
          <Ontology.Tree root={group.data} />
        </Align.Space>
      </RackHeartbeatProvider>
    </StateProvider>
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

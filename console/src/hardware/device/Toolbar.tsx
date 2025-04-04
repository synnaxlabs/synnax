import { type device, ontology, type rack } from "@synnaxlabs/client";
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
  states: Record<string, rack.State>;
}

const RackStateContext = createContext<RackHeartbeatProviderContextValue>({
  states: {},
});

const RackHeartbeatProvider = ({ children }: { children: ReactElement }) => {
  const client = Synnax.use();
  const [states, setStates] = reactUseState<Record<string, rack.State>>({});

  useAsyncEffect(async () => {
    if (client == null) return;
    const racks = await client.hardware.racks.retrieve([], {
      includeState: true,
    });
    const initialStates = Object.fromEntries(
      racks.filter((r) => r.state != null).map((r) => [r.key, r.state]),
    ) as Record<string, rack.State>;
    setStates(initialStates);
    const observer = await client.hardware.racks.openStateObserver();
    const disconnect = observer.onChange((states) => {
      setStates((prev) => {
        const newStates = Object.fromEntries(states.map((s) => [s.key, s]));
        return { ...prev, ...newStates };
      });
    });
    return async () => {
      disconnect();
      await observer.close();
    };
  }, []);

  return (
    <RackStateContext.Provider value={{ states }}>{children}</RackStateContext.Provider>
  );
};

export const useRackState = (key: string): rack.State | undefined => {
  const { states } = useContext(RackStateContext);
  return states[key];
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

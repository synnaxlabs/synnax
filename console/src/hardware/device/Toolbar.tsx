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

export interface ContextValue {
  states: Record<string, device.State>;
}

const Context = createContext<ContextValue>({
  states: {},
});

const Provider = ({ children }: { children: ReactElement }) => {
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

  return <Context.Provider value={{ states }}>{children}</Context.Provider>;
};

export const useState = (key: string): device.State | undefined => {
  const { states } = useContext(Context);
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
    <Provider>
      <Align.Space empty style={{ height: "100%" }}>
        <Toolbar.Header>
          <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
          <Header.Actions></Header.Actions>
        </Toolbar.Header>
        <Ontology.Tree root={group.data} />
      </Align.Space>
    </Provider>
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

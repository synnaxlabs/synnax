import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Header, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { Toolbar } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

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
    <Align.Space empty style={{ height: "100%" }}>
      <Toolbar.Header>
        <Toolbar.Title icon={<Icon.Device />}>Devices</Toolbar.Title>
        <Header.Actions></Header.Actions>
      </Toolbar.Header>
      <Ontology.Tree root={group.data} />
    </Align.Space>
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

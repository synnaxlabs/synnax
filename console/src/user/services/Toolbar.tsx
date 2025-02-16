import { ontology } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Header, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery<ontology.ID | null>({
    queryKey: [client?.key, "user-group"],
    queryFn: async () => {
      if (client == null) return null;
      const res = await client?.ontology.retrieveChildren(ontology.ROOT_ID, {
        includeSchema: false,
      });
      return res?.filter((r) => r.name === "Users")[0].id;
    },
  });

  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.User />}>Users</ToolbarTitle>
        <Header.Actions></Header.Actions>
      </ToolbarHeader>
      <Ontology.Tree root={group.data} />
    </Align.Space>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "user",
  icon: <Icon.User />,
  content: <Content />,
  tooltip: "Users",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};

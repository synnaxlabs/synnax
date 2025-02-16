import { type group } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/media";
import { Align, Header, Synnax } from "@synnaxlabs/pluto";
import { useQuery } from "@tanstack/react-query";
import { type ReactElement } from "react";

import { ToolbarHeader, ToolbarTitle } from "@/components";
import { type Layout } from "@/layout";
import { Ontology } from "@/ontology";
import { CREATE_LAYOUT } from "@/range/Create";

const Content = (): ReactElement => {
  const client = Synnax.use();
  const group = useQuery<group.Group | null>({
    queryKey: [client?.key, "channel-group"],
    queryFn: async () => {
      if (client == null) return null;
      return await client?.channels.retrieveGroup();
    },
  });

  console.log(group.data);

  return (
    <Align.Space empty style={{ height: "100%" }}>
      <ToolbarHeader>
        <ToolbarTitle icon={<Icon.Channel />}>Channels</ToolbarTitle>
        <Header.Actions>
          {[{ children: <Icon.Add />, onClick: () => placeLayout(CREATE_LAYOUT) }]}
        </Header.Actions>
      </ToolbarHeader>
      <Ontology.Tree root={group.data?.ontologyID} />
    </Align.Space>
  );
};

export const TOOLBAR: Layout.NavDrawerItem = {
  key: "channel",
  icon: <Icon.Channel />,
  content: <Content />,
  tooltip: "Channels",
  initialSize: 300,
  minSize: 175,
  maxSize: 400,
};

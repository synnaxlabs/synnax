import { type ReactElement } from "react";

import { type hardware } from "@synnaxlabs/client";
import { Synnax } from "@synnaxlabs/pluto";
import { List } from "@synnaxlabs/pluto/list";
import { useQuery } from "@tanstack/react-query";

export const RackList = (): ReactElement => {
  const client = Synnax.use();

  useQuery({
    queryKey: ["racks", client?.createdAt.toString()],
    queryFn: async () => {
      if (client == null) return;
      await client.hardware.racks.retrieve();
    },
  });

  return (
    <List.List>
      <List.Core>{(p) => <RackListItem {...p} />}</List.Core>
    </List.List>
  );
};

interface RackListItemProps extends List.ItemProps<hardware.rack.{
  : hardware.rack.Rack;
  
}

const RackListItem = () => {};

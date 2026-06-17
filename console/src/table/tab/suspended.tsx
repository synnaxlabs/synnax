import { Table } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { useArgs } from "@/table/tab/useArgs";

export const createSuspended = <P extends {} = {}>(Internal: FC<P>) => {
  const Suspended = (props: P) => {
    const { key } = useArgs();
    return (
      <Table.Suspended tableKey={key}>
        <Internal {...props} />
      </Table.Suspended>
    );
  };
  return Suspended;
};

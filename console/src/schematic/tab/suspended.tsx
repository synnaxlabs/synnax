import { Schematic } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { useArgs } from "@/schematic/tab/useArgs";

export const createSuspended = <P extends {} = {}>(Internal: FC<P>) => {
  const Suspended = (props: P) => {
    const { key } = useArgs();
    return (
      <Schematic.Suspended schematicKey={key}>
        <Internal {...props} />;
      </Schematic.Suspended>
    );
  };
  return Suspended;
};

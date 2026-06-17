import { LinePlot } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { useArgs } from "@/lineplot/tab/useArgs";

export const createSuspended = <P extends {} = {}>(Internal: FC<P>) => {
  const Suspended = (props: P) => {
    const { key } = useArgs();
    return (
      <LinePlot.Suspended lineplotKey={key}>
        <Internal {...props} />;
      </LinePlot.Suspended>
    );
  };
  return Suspended;
};

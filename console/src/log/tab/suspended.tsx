import { Log } from "@synnaxlabs/pluto";
import { type FC } from "react";

import { useArgs } from "@/log/tab/useArgs";

export const createSuspended = <P extends {} = {}>(Internal: FC<P>) => {
  const Suspended = (props: P) => {
    const { key } = useArgs();
    return (
      <Log.Suspended logKey={key}>
        <Internal {...props} />
      </Log.Suspended>
    );
  };
  return Suspended;
};

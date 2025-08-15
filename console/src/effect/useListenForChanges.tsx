import { effect, TimeStamp } from "@synnaxlabs/client";
import { Flux, Status } from "@synnaxlabs/pluto";
import { useMemo } from "react";

export const useListenForChanges = () => {
  const addStatus = Status.useAdder();
  const handler = useMemo(
    () =>
      Flux.parsedHandler(effect.statusZ, async ({ changed }) => {
        addStatus({
          ...changed,
          time: TimeStamp.now(),
        });
      }),
    [addStatus],
  );
  Flux.useListener({
    channel: effect.STATUS_CHANNEL_NAME,
    onChange: handler,
  });
};

import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

export const useListenForChanges = () => {
  const addStatus = Status.useAdder();
  const handleSet = useCallback(addStatus, [addStatus]);
  Status.useSetSynchronizer(handleSet);
};

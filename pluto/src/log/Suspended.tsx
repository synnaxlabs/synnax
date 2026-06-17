import { type log } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";
import { useEnsureRetrieved } from "@/log/queries";

const [KeyContext, useKeyBase] = context.create<log.Key>({
  displayName: "KeyContext.Provider",
  providerName: "KeyContext.Provider",
});

export const useKey = () => useKeyBase("useKey");

export interface SuspendedProps extends PropsWithChildren {
  logKey: log.Key;
}

export const Suspended = ({ logKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: logKey });
  return <KeyContext value={logKey}>{children}</KeyContext>;
};

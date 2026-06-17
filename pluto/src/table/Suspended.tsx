import { type table } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";
import { useEnsureRetrieved } from "@/table/queries";

const [KeyContext, useKeyBase] = context.create<table.Key>({
  displayName: "KeyContext.Provider",
  providerName: "KeyContext.Provider",
});

export const useKey = () => useKeyBase("useKey");

export interface SuspendedProps extends PropsWithChildren {
  tableKey: table.Key;
}

export const Suspended = ({ tableKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: tableKey });
  return <KeyContext value={tableKey}>{children}</KeyContext>;
};

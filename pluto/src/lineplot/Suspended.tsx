import { type lineplot, type schematic } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";
import { useEnsureRetrieved } from "@/lineplot/queries";

const [KeyContext, useKeyBase] = context.create<schematic.Key>({
  displayName: "KeyContext.Provider",
  providerName: "KeyContext.Provider",
});

export const useKey = () => useKeyBase("cat");

export interface SuspendedProps extends PropsWithChildren {
  lineplotKey: lineplot.Key;
}

export const Suspended = ({ lineplotKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: lineplotKey });
  return <KeyContext value={lineplotKey}>{children}</KeyContext>;
};

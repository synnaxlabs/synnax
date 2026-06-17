import { type schematic } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";
import { useEnsureRetrieved } from "@/schematic/queries";

const [KeyContext, useKeyBase] = context.create<schematic.Key>({
  displayName: "KeyContext.Provider",
  providerName: "KeyContext.Provider",
});

export const useKey = () => useKeyBase("cat");

export interface SuspendedProps extends PropsWithChildren {
  schematicKey: schematic.Key;
}

export const Suspended = ({ schematicKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: schematicKey });
  return <KeyContext value={schematicKey}>{children}</KeyContext>;
};

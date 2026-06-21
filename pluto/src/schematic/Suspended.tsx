import { NotFoundError, type schematic } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";

import { useEnsureRetrieved } from "./queries";

const [KeyContext, useOptionalKey] = context.create<schematic.Key | undefined>({
  displayName: "KeyContext.Provider",
  defaultValue: undefined,
});

export { useOptionalKey };

export const useKey = (): schematic.Key => {
  const value = useOptionalKey();
  if (value == null)
    throw new NotFoundError(
      `Schematic.useKey must be called inside of Schematic.KeyContext`,
    );
  return value;
};

export interface SuspendedProps extends PropsWithChildren {
  schematicKey: schematic.Key;
}

export const Suspended = ({ schematicKey, children }: SuspendedProps): ReactElement => {
  useEnsureRetrieved({ key: schematicKey });
  return <KeyContext value={schematicKey}>{children}</KeyContext>;
};

import { NotFoundError, type schematic } from "@synnaxlabs/client";
import { type PropsWithChildren, type ReactElement } from "react";

import { context } from "@/context";

import { useEnsureRetrieved } from "./queries";

const [KeyContext, useOptionalKey] = context.create<schematic.Key | undefined>({
  displayName: "KeyContext.Provider",
  defaultValue: undefined,
});

export const useKey = (override?: schematic.Key): schematic.Key => {
  const value = useOptionalKey() ?? override;
  if (value == null)
    throw new NotFoundError(
      `Schematic.useKey must be passed an override or called inside of Schematic.KeyContext`,
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

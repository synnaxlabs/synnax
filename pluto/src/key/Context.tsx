import { type record } from "@synnaxlabs/x";
import { type PropsWithChildren } from "react";

import { context } from "@/context";

export const [BaseProvider, baseUse] = context.create<record.Key>({
  displayName: "Key.Context",
  providerName: "Key.Provider",
});

export const use = <K extends record.Key>(hookOrComponentName: string): K =>
  baseUse(hookOrComponentName) as K;

export interface ProviderProps<K extends record.Key> extends PropsWithChildren {
  value: K;
}

export const Provider = <K extends record.Key>(props: ProviderProps<K>) => (
  <Provider {...props} />
);

import { type bounds, type box } from "@synnaxlabs/x";

import { context } from "@/context";

export interface Entry<T extends number | bigint = number> {
  key: string;
  bounds: bounds.Bounds<T>;
}

export interface ContextValue<T extends number | bigint = number | bigint> {
  bounds: bounds.Bounds<T>;
  setEntry: (entry: Entry<T>) => void;
  viewport: box.Box;
  setViewport: (viewport: box.Box) => void;
}

export const [Context, useContext] = context.create<ContextValue>({
  displayName: "Timeline.Context",
  providerName: "Timeline.Provider",
});

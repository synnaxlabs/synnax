import { VersionStoreState } from "./slice";

import { useMemoSelect } from "@/hooks";

export const selectVersion = (state: VersionStoreState): string =>
  state.version.version;

export const useSelectVersion = (): string => useMemoSelect(selectVersion, []);

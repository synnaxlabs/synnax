import { useCallback } from "react";

import { type Layout } from "@/layout";

type UpdateFn = (params: { key: string; name: string }) => void;
type RetrieveFn = (params: { key: string }) => void;

export const createFluxUseName =
  (
    useRename: () => { update: UpdateFn },
    useRetrieve: (params: {
      onChange: (name: string) => void;
      addStatusOnFailure: boolean;
    }) => { retrieve: RetrieveFn },
    useRemoteCreated?: (layoutKey: string) => boolean | undefined,
  ): Layout.NameHook =>
  (layoutKey, onChange) => {
    const isRemote = useRemoteCreated?.(layoutKey) === true;
    const { retrieve: baseRetrieve } = useRetrieve({
      onChange,
      addStatusOnFailure: false,
    });
    const { update } = useRename();
    const onRename = useCallback(
      (name: string) => isRemote && update({ key: layoutKey, name }),
      [layoutKey, update, isRemote],
    );
    const retrieve = useCallback(
      () => isRemote && baseRetrieve({ key: layoutKey }),
      [layoutKey, baseRetrieve, isRemote],
    );
    return { retrieve, onRename };
  };

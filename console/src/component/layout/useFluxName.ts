// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { type UseName } from "@/component/layout/layout";

type UpdateFn = (params: { key: string; name: string }) => void;
type RetrieveFn = (params: { key: string }) => void;

export const createUseFluxName =
  (
    useRename: () => { update: UpdateFn },
    useRetrieve: (params: {
      onChange: (name: string) => void;
      addStatusOnFailure: boolean;
    }) => { retrieve: RetrieveFn },
    useEnabled?: (layoutKey: string) => boolean | undefined,
  ): UseName =>
  (layoutKey, onChange) => {
    let isEnabled = true;
    // It's safe to call this hook conditionally as it is passed in as part
    // of a static factory.
    if (useEnabled != null) isEnabled = useEnabled(layoutKey) === true;
    const { retrieve: baseRetrieve } = useRetrieve({
      onChange,
      addStatusOnFailure: false,
    });
    const { update } = useRename();
    const onRename = useCallback(
      (name: string) => isEnabled && update({ key: layoutKey, name }),
      [layoutKey, update, isEnabled],
    );
    const retrieve = useCallback(
      () => isEnabled && baseRetrieve({ key: layoutKey }),
      [layoutKey, baseRetrieve, isEnabled],
    );
    return { retrieve, onRename };
  };

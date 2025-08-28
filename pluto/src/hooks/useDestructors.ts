// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { array, type Destructor } from "@synnaxlabs/x";
import { useEffect, useMemo, useRef } from "react";

export interface UseDestructorsReturn {
  cleanup: () => void;
  set: (destructors: Destructor | Destructor[] | undefined) => void;
}

export const useDestructors = (): UseDestructorsReturn => {
  const ref = useRef<Destructor[]>([]);
  const value = useMemo(
    () => ({
      cleanup: () => {
        ref.current.forEach((destructor) => destructor());
        ref.current = [];
      },
      set: (destructors: Destructor | Destructor[] | undefined): void => {
        if (destructors == null) return;
        ref.current.push(...array.toArray(destructors));
      },
    }),
    [],
  );
  useEffect(() => value.cleanup, [value.cleanup]);
  return value;
};

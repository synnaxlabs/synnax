// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, useCallback } from "react";

import { box } from "@synnaxlabs/x";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useResize, useSyncedRef } from "@/hooks";
import { eraser } from "@/vis/eraser/aether";

export interface UseProps {
  aetherKey: string;
}

export const use = ({ aetherKey }: UseProps): ((region: box.Box) => void) => {
  const [, { region }, setState] = Aether.use({
    aetherKey,
    type: eraser.Eraser.TYPE,
    schema: eraser.eraserStateZ,
    initialState: { region: box.ZERO },
  });
  const regionRef = useSyncedRef(region);
  return useCallback(
    (b: box.Box) => {
      if (box.equals(b, regionRef.current)) return;
      setState((p) => ({ ...p, region: b }));
    },
    [setState],
  );
};

export interface EraserProps extends PropsWithChildren {}

export const Eraser = Aether.wrap<EraserProps>(
  eraser.Eraser.TYPE,
  ({ aetherKey, children }) => {
    const erase = use({ aetherKey });
    const ref = useResize(erase);
    return (
      <div ref={ref} className={CSS(CSS.inheritDims())}>
        {children}
      </div>
    );
  },
);

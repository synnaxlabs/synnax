// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x";
import { type PropsWithChildren, useCallback } from "react";

import { Aether } from "@/aether";
import { CSS } from "@/css";
import { useResize, useSyncedRef } from "@/hooks";
import { eraser } from "@/vis/eraser/aether";

export interface UseProps {
  aetherKey: string;
}

export interface UseReturn {
  setEnabled: (enabled: boolean) => void;
  erase: (region: box.Box) => void;
}

export const use = ({ aetherKey }: UseProps): UseReturn => {
  const [, { region }, setState] = Aether.use({
    aetherKey,
    type: eraser.Eraser.TYPE,
    schema: eraser.eraserStateZ,
    initialState: { region: box.ZERO, enabled: true },
  });
  const regionRef = useSyncedRef(region);
  const erase = useCallback(
    (b: box.Box) => {
      if (box.equals(b, regionRef.current)) return;
      setState((p) => ({ ...p, region: b }));
    },
    [setState],
  );
  const setEnabled = useCallback(
    (enabled: boolean) => setState((p) => ({ ...p, enabled })),
    [setState],
  );

  return { setEnabled, erase };
};

export interface EraserProps extends PropsWithChildren {}

export const Eraser = Aether.wrap<EraserProps>(
  eraser.Eraser.TYPE,
  ({ aetherKey, children }) => {
    const { erase } = use({ aetherKey });
    const ref = useResize(erase);
    return (
      <div ref={ref} className={CSS(CSS.inheritDims())}>
        {children}
      </div>
    );
  },
);

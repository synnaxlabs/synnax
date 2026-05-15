// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box } from "@synnaxlabs/x/box";
import { Aether } from "@synnaxlabs/lyra/aether";
import { CSS } from "@synnaxlabs/lyra/css";
import { useResize, useSyncedRef } from "@synnaxlabs/lyra/hooks";

import { type PropsWithChildren, type ReactElement, useCallback } from "react";

import { eraser } from "@/vis/eraser/aether";

export interface UseProps {
  aetherKey?: string;
  enabled?: boolean;
}

export interface UseReturn {
  erase: (region: box.Box) => void;
}

export const use = ({ aetherKey, enabled = true }: UseProps): UseReturn => {
  const { setState } = Aether.useLifecycle({
    aetherKey,
    type: eraser.Eraser.TYPE,
    schema: eraser.eraserStateZ,
    initialState: { region: box.ZERO, enabled },
  });
  const enabledRef = useSyncedRef(enabled);
  const erase = useCallback(
    (b: box.Box) => {
      setState({ enabled: enabledRef.current, region: b });
    },
    [setState, enabledRef],
  );
  return { erase };
};

export interface EraserProps extends PropsWithChildren, Aether.ComponentProps {}

export const Eraser = ({ aetherKey, children }: EraserProps): ReactElement => {
  const { erase } = use({ aetherKey });
  const ref = useResize(erase);
  return (
    <div ref={ref} className={CSS(CSS.inheritDims())}>
      {children}
    </div>
  );
};

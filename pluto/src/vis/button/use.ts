// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useRef } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { button } from "@/vis/button/aether";

export type Mode = button.Mode;
export const MODES = button.MODES;

export interface UseProps extends Omit<z.input<typeof button.buttonStateZ>, "trigger"> {
  aetherKey: string;
}

export interface UseReturn {
  onClick: () => void;
  onMouseDown: () => void;
  onMouseUp: () => void;
}

export const use = ({ aetherKey, sink, mode }: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ sink, mode });
  const propsRef = useRef({ trigger: 0, ...memoProps });
  const { setState } = Aether.useLifecycle({
    aetherKey,
    type: button.Button.TYPE,
    schema: button.buttonStateZ,
    initialState: propsRef.current,
  });

  useEffect(() => {
    propsRef.current = { ...propsRef.current, ...memoProps };
    setState(propsRef.current);
  }, [memoProps, setState, aetherKey]);

  const onMouseUp = useCallback(() => {
    propsRef.current.trigger += button.MOUSE_UP_INCREMENT;
    setState(propsRef.current);
  }, [setState]);

  const onMouseDown = useCallback(() => {
    propsRef.current.trigger += button.MOUSE_DOWN_INCREMENT;
    setState(propsRef.current);
  }, [setState]);

  return { onClick: onMouseUp, onMouseDown, onMouseUp };
};

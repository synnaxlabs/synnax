/* eslint-disable @typescript-eslint/explicit-function-return-type */
// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect } from "react";

import { mouseButtonKey } from "./mouse";
import { KeyboardKey } from "./types";

import { PseudoSetState } from "@/hooks/useStateRef";
import { compareArrayDeps, useMemoCompare } from "@/memo";

export const useKeysHeld = (
  onChange: PseudoSetState<KeyboardKey[]>,
  ..._keys: KeyboardKey[]
): void => {
  const keys = useMemoCompare(() => _keys, compareArrayDeps, [_keys] as const);

  const handlePress = useCallback(
    (key: KeyboardKey) => onChange((prev) => [...prev, key]),
    [keys]
  );

  const handleRelease = useCallback(
    (key: KeyboardKey) => onChange((prev) => prev.filter((k) => k !== key)),
    [keys]
  );

  useKeyPress({
    keys,
    onPress: handlePress,
    onRelease: handleRelease,
  });
};

interface KeyPressEvent {
  key: KeyboardKey;
}

export const useKeyPress = ({
  keys,
  onPress,
  onRelease,
}: {
  keys: KeyboardKey[];
  onPress?: (key: KeyboardKey) => void;
  onRelease?: (key: KeyboardKey) => void;
}): void => {
  useEffect(() => {
    const onKeyDown = (e: KeyPressEvent) => keys.includes(e.key) && onPress?.(e.key);
    const onKeyUp = (e: KeyPressEvent) => keys.includes(e.key) && onRelease?.(e.key);
    const onMouseDown = (e: MouseEvent) => onKeyDown({ key: mouseButtonKey(e) });
    const onMouseUp = (e: MouseEvent) => onKeyUp({ key: mouseButtonKey(e) });
    window.addEventListener("keydown", onKeyDown);
    window.addEventListener("keyup", onKeyUp);
    window.addEventListener("mousedown", onMouseDown);
    window.addEventListener("mouseup", onMouseUp);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("keyup", onKeyUp);
      window.removeEventListener("mousedown", onMouseDown);
      window.removeEventListener("mouseup", onMouseUp);
    };
  }, [keys, onPress, onRelease]);
};

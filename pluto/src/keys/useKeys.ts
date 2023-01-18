// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState } from "react";

import { compareArrayDeps, useMemoCompare } from "..";

import { mouseDownToKey as mouseClickAsKey } from "./mouse";
import { KeyboardKey } from "./types";

export interface UseKeyHeldReturn {
  keys: KeyboardKey[];
  any: boolean;
  all: boolean;
}

export const useKeysHeld = (..._keys: KeyboardKey[]): UseKeyHeldReturn => {
  const [state, setState] = useState<KeyboardKey[]>([]);

  const keys = useMemoCompare(() => _keys, compareArrayDeps, [_keys] as const);

  const handlePress = useCallback(
    (key: KeyboardKey) => setState((prev) => [...prev, key]),
    [keys]
  );

  const handleRelease = useCallback(
    (key: KeyboardKey) => setState((prev) => prev.filter((k) => k !== key)),
    [keys]
  );

  useKeyPress({
    keys,
    onPress: handlePress,
    onRelease: handleRelease,
  });

  return {
    keys: state,
    any: state.length > 0,
    all: state.length === _keys.length,
  };
};

export interface KeyPressEvent {
  key: KeyboardKey;
}

export const useKeyPress = ({
  keys,
  onPress,
  onRelease,
}: {
  keys: KeyboardKey[] | null;
  onPress: (key: KeyboardKey) => void;
  onRelease?: (key: KeyboardKey) => void;
}): void => {
  useEffect(() => {
    const onKeyDown = (e: KeyPressEvent): void => {
      if (keys == null || keys.includes(e.key)) onPress(e.key);
    };
    const onKeyUp = (e: KeyPressEvent): void => {
      if (keys == null || keys.includes(e.key)) onRelease?.(e.key);
    };
    const onMouseDown = (e: MouseEvent): void => onKeyDown({ key: mouseClickAsKey(e) });
    const onMouseUp = (e: MouseEvent): void => onKeyUp({ key: mouseClickAsKey(e) });
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

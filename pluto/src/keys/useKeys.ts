// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect, useState } from "react";

import { KeyboardKey } from "./types";

export interface UseKeyHeldReturn {
  keys: KeyboardKey[];
  any: boolean;
  all: boolean;
}

export const useKeysHeld = (keys: KeyboardKey[]): UseKeyHeldReturn => {
  const [held, setHeld] = useState<UseKeyHeldReturn>({
    keys: [],
    any: false,
    all: false,
  });

  const handlePress = useCallback(
    (key: KeyboardKey) =>
      setHeld((prev) => {
        if (keys == null) return prev;
        const nextKeys = [...prev.keys, key];
        return {
          keys: nextKeys,
          any: true,
          all: nextKeys.length === keys.length,
        };
      }),
    [keys]
  );

  const handleRelease = useCallback(
    (key: KeyboardKey) =>
      setHeld((prev) => {
        if (keys == null) return prev;
        const nextKeys = prev.keys.filter((k) => k !== key);
        return {
          keys: nextKeys,
          any: nextKeys.length > 0,
          all: nextKeys.length === keys.length,
        };
      }),
    [keys]
  );

  useKeyPress({
    keys,
    onPress: handlePress,
    onRelease: handleRelease,
  });
  if (keys == null) return { keys: [], any: true, all: true };
  return held;
};

export const useKeyPress = ({
  keys,
  onPress,
  onRelease,
}: {
  keys: KeyboardKey[] | null;
  onPress: (key: KeyboardKey) => void;
  onRelease?: (key: KeyboardKey) => void;
}): void =>
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent): void => {
      console.log(e.key);
      if (keys == null || keys.includes(e.key)) onPress(e.key);
    };
    const onKeyUp = (e: KeyboardEvent): void => {
      if (keys == null || keys.includes(e.key)) onRelease?.(e.key);
    };
    window.addEventListener("keydown", onKeyDown, true);
    window.addEventListener("keyup", onKeyUp, true);
    return () => {
      window.removeEventListener("keydown", onKeyDown);
      window.removeEventListener("keyup", onKeyUp);
    };
  }, [keys, onPress, onRelease]);

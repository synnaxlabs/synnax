// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

export const useKeyHeld = (key: string): boolean => {
  const [held, setHeld] = useState(false);
  useKeyPress({
    key,
    onPress: () => setHeld(true),
    onRelease: () => setHeld(false),
  });
  return held;
};

export const useKeyPress = ({
  key,
  onPress,
  onRelease,
}: {
  key: string;
  onPress: () => void;
  onRelease?: () => void;
}): void => {
  useEffect(() => {
    const onKeyDown = (e: KeyboardEvent): void => {
      if (e.key === key) onPress();
    };
    const onKeyUp = (e: KeyboardEvent): void => {
      if (e.key === key) onRelease?.();
    };
    document.addEventListener("keydown", onKeyDown);
    document.addEventListener("keyup", onKeyUp);
    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.removeEventListener("keyup", onKeyUp);
    };
  });
};

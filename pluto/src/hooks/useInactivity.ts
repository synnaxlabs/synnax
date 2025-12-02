// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { throttle } from "@synnaxlabs/x";
import { type RefObject, useCallback, useEffect, useRef, useState } from "react";
export interface UseInactivityReturn<E extends HTMLElement> {
  visible: boolean;
  ref: RefObject<E | null>;
}

export const useInactivity = <E extends HTMLElement>(
  timeout: number,
): UseInactivityReturn<E> => {
  const [visible, setVisible] = useState(false);
  const inactivityTimeoutRef = useRef<NodeJS.Timeout>(undefined);
  const ref = useRef<E>(null);

  const handleMouseMove = useCallback(
    throttle(() => {
      setVisible(true);
      clearTimeout(inactivityTimeoutRef.current);
      inactivityTimeoutRef.current = setTimeout(() => setVisible(false), timeout);
    }, 150),
    [timeout],
  );

  const handleMouseLeave = useCallback(() => {
    setVisible(false);
    clearTimeout(inactivityTimeoutRef.current);
  }, []);

  useEffect(() => {
    const el = ref.current;
    if (el == null) return;
    el.addEventListener("mousemove", handleMouseMove);
    el.addEventListener("mouseleave", handleMouseLeave);
    return () => {
      el.removeEventListener("mousemove", handleMouseMove);
      el.removeEventListener("mouseleave", handleMouseLeave);
      clearTimeout(inactivityTimeoutRef.current);
    };
  }, [handleMouseLeave, handleMouseMove]);

  return { visible, ref };
};

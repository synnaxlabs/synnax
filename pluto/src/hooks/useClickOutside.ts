// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { box, xy } from "@synnaxlabs/x";
import { type RefObject, useCallback, useEffect } from "react";

import { useSyncedRef } from "@/hooks/ref";

export interface UseClickOutsideProps {
  ref: RefObject<HTMLElement | null>;
  exclude?: Array<RefObject<HTMLElement>> | ((e: MouseEvent) => boolean);
  onClickOutside: () => void;
}

/**
 * A hooks that calls the provided callback when a click event occurs outside of the
 * provided ref.
 * @param ref - The ref to check for clicks outside of.
 * @param onClickOutside - The callback to call when a click event occurs outside of the
 * provided ref.
 */
export const useClickOutside = ({
  ref,
  onClickOutside,
  exclude,
}: UseClickOutsideProps): void => {
  const excludeRef = useSyncedRef(exclude);
  const handleClickOutside = useCallback(
    (e: MouseEvent): void => {
      const el = ref.current;
      const windowBox = box.construct(window.document.documentElement);
      const pos = xy.construct(e);

      const exclude = excludeRef.current;
      if (exclude != null)
        if (typeof exclude === "function") {
          if (exclude(e)) return;
        } else if (exclude.some((r) => r.current?.contains(e.target as Node))) return;

      if (
        el == null ||
        el.contains(e.target as Node) ||
        box.contains(el, pos) ||
        !box.contains(windowBox, pos)
      )
        return;
      onClickOutside();
    },
    [onClickOutside],
  );
  useEffect(() => {
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [handleClickOutside]);
};

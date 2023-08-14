// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RefObject, useCallback, useEffect } from "react";

/**
 * A hooks that calls the provided callback when a click event occurs outside of the
 * provided ref.
 * @param ref - The ref to check for clicks outside of.
 * @param onClickOutside - The callback to call when a click event occurs outside of the
 * provided ref.
 */
export const useClickOutside = (
  ref: RefObject<HTMLElement>,
  onClickOutside: () => void
): void => {
  const handleClickOutside = useCallback(
    ({ target }: MouseEvent): void => {
      const el = ref.current;
      if (el == null || el.contains(target as Node)) return;
      onClickOutside();
    },
    [onClickOutside]
  );
  useEffect(() => {
    document.addEventListener("mousedown", handleClickOutside);
    window.addEventListener("blur", onClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [handleClickOutside]);
};

// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { RefObject, useEffect } from "react";

export const useClickoutside = (
  ref: RefObject<HTMLElement>,
  onClickOutside: () => void
): void =>
  useEffect(() => {
    const { current: el } = ref;
    const handleClickOutside = ({ target }: MouseEvent): void => {
      if (el != null && !el.contains(target as Node)) onClickOutside();
    };
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, [ref, onClickOutside]);

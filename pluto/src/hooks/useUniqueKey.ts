// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { customAlphabet } from "nanoid/non-secure";
import { useRef } from "react";

// NOTE: very important to not add characters to this alphabet that can cause issues
// with CSS.
const nanoid = customAlphabet(
  "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZ_abcdefghijklmnopqrstuvwxyz-",
  9,
);

/**
 * @returns a unique key that persists for the lifetime of the component.
 * @param override - An optional override for the key. If provided, this will be returned
 * instead of the generated key.
 * */
export const useUniqueKey = (override?: string): string => {
  const gen = useRef<string | null>(null);
  if (gen.current === null) gen.current = override ?? nanoid();
  return gen.current;
};

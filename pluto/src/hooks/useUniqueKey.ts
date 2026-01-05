// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { id } from "@synnaxlabs/x";
import { useRef } from "react";

/**
 * @returns a unique key that persists for the lifetime of the component.
 * @param override - An optional override for the key. If provided, this will be
 * returned instead of the created key.
 * */
export const useUniqueKey = (override?: string): string => {
  const gen = useRef<string | null>(null);
  if (gen.current === null) gen.current = override ?? id.create();
  return gen.current;
};

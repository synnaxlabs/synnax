// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { TimeSpan } from "@synnaxlabs/x";
import { useEffect, useState } from "react";

// A check against a dead local port fails in milliseconds, too fast to see.
const HOLD = TimeSpan.milliseconds(1250);

/**
 * Whether a connection check is in flight, held after a fast one so it stays
 * legible. Read this instead of `details.checking` for anything the user looks
 * at, and read `details.checking` for anything the user acts on.
 */
export const useHeldChecking = (checking: boolean): boolean => {
  const [held, setHeld] = useState(checking);
  useEffect(() => {
    if (checking) {
      setHeld(true);
      return;
    }
    const timeout = setTimeout(() => setHeld(false), HOLD.milliseconds);
    return () => clearTimeout(timeout);
  }, [checking]);
  return held;
};

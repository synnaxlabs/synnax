// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { languageLoaded, loadLanguage } from "@/input/time/suggest";

/**
 * Loads the natural-language parser on mount.
 * @returns true once phrases read, so a memoized suggest function can refresh.
 */
export const useLanguage = (): boolean => {
  const [ready, setReady] = useState(languageLoaded);
  useEffect(() => {
    if (ready) return;
    void loadLanguage().then(() => setReady(true));
  }, [ready]);
  return ready;
};

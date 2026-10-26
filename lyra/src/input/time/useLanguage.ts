// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect, useState } from "react";

import { type Language, loadedLanguage, loadLanguage } from "@/input/time/suggest";

/**
 * Loads the natural-language parser on mount.
 * @returns the parser once it loads, or null until then.
 */
export const useLanguage = (): Language | null => {
  // The parser is callable, so it passes through updaters, never as a bare value.
  const [language, setLanguage] = useState(() => loadedLanguage());
  useEffect(() => {
    if (language != null) return;
    void loadLanguage().then((loaded) => setLanguage(() => loaded));
  }, [language]);
  return language;
};

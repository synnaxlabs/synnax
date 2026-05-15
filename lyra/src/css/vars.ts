// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const applyCSSVars = (
  element: HTMLElement,
  vars: Record<string, string | number | undefined>,
): void =>
  Object.entries(vars).forEach(
    ([key, value]) => value != null && element.style.setProperty(key, `${value}`),
  );

export const removeCSSVars = (element: HTMLElement, prefix: string): void =>
  Array.from(element.style).forEach((key) => {
    if (key.startsWith(prefix)) element.style.removeProperty(key);
  });

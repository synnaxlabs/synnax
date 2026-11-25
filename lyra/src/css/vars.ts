// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

// A session recorder re-reads the whole style attribute on each mutation, so one write
// per variable makes a theme change take seconds.
const editStyle = (
  element: HTMLElement,
  edit: (style: CSSStyleDeclaration) => void,
): void => {
  const draft = element.ownerDocument.createElement("div");
  draft.style.cssText = element.style.cssText;
  edit(draft.style);
  element.style.cssText = draft.style.cssText;
};

/** Sets each defined variable in vars on the element's inline style in one write. */
export const applyCSSVars = (
  element: HTMLElement,
  vars: Record<string, string | number | undefined>,
): void =>
  editStyle(element, (style) =>
    Object.entries(vars).forEach(
      ([key, value]) => value != null && style.setProperty(key, `${value}`),
    ),
  );

/** Removes each inline style property that starts with prefix in one write. */
export const removeCSSVars = (element: HTMLElement, prefix: string): void =>
  editStyle(element, (style) =>
    Array.from(style).forEach((key) => {
      if (key.startsWith(prefix)) style.removeProperty(key);
    }),
  );

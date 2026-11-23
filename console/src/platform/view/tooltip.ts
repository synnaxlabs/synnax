// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/**
 * @returns a `hideTooltip` check that hides a control's tooltip while the control shows
 * its text, the element with the given class.
 */
export const textShown =
  (className: string) =>
  (anchor: HTMLElement): boolean => {
    const text = anchor.querySelector(`.${className}`);
    return text != null && getComputedStyle(text).display !== "none";
  };

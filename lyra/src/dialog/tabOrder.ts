// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { FOCUSABLE } from "@/util";

/** Whether b comes after a in the document. */
export const follows = (a: Node, b: Node): boolean =>
  (a.compareDocumentPosition(b) & Node.DOCUMENT_POSITION_FOLLOWING) !== 0;

/** Whether Tab can stop on the element. */
export const isTabbable = (el: HTMLElement): boolean =>
  el.matches(FOCUSABLE) &&
  el.tabIndex >= 0 &&
  !el.matches(":disabled") &&
  el.getClientRects().length > 0;

/** @returns the elements in root that Tab can stop on, in document order. */
export const tabbable = (root: HTMLElement): HTMLElement[] =>
  Array.from(root.querySelectorAll<HTMLElement>(FOCUSABLE)).filter(isTabbable);

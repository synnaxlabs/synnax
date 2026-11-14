// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Matches the elements that can take the focus, enabled or not. */
export const FOCUSABLE =
  'a[href], button, input, select, textarea, [contenteditable="true"], [tabindex]';

/**
 * Reports whether the element that handles an event owns it: the event starts inside
 * its DOM tree, and no focusable descendant takes it. An event that React bubbles up
 * from a portal fails.
 */
export const ownsEvent = ({
  target,
  currentTarget,
}: {
  target: EventTarget;
  currentTarget: Element;
}): boolean => {
  if (!(target instanceof Element) || !currentTarget.contains(target)) return false;
  const owner = target.closest(FOCUSABLE);
  return owner == null || owner === currentTarget || !currentTarget.contains(owner);
};

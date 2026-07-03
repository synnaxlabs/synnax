// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

/** Queries container for a rendered pluto icon by its name, or null. */
export const queryIcon = (container: ParentNode, icon: string): Element | null =>
  container.querySelector(`[aria-label="pluto-icon--${icon}"]`);

/** Returns the closest button wrapping the given pluto icon, or null. */
export const queryIconButton = (
  container: ParentNode,
  icon: string,
): HTMLButtonElement | null => queryIcon(container, icon)?.closest("button") ?? null;

/** Like queryIconButton, but throws when no button wraps the icon. */
export const getIconButton = (
  container: ParentNode,
  icon: string,
): HTMLButtonElement => {
  const button = queryIconButton(container, icon);
  if (button == null) throw new Error(`no button wrapping icon ${icon}`);
  return button;
};

/** Queries container for the given selector, throwing when nothing matches. */
export const getBySelector = <E extends Element = Element>(
  container: ParentNode,
  selector: string,
): E => {
  const el = container.querySelector<E>(selector);
  if (el == null) throw new Error(`no element matches selector ${selector}`);
  return el;
};

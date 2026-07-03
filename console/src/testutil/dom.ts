// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { screen } from "@testing-library/react";

/** Queries container for a rendered pluto icon by its name, or null. */
export const queryIcon = (container: ParentNode, icon: string): Element | null =>
  container.querySelector(`[aria-label="pluto-icon--${icon}"]`);

/** Returns the closest button wrapping the given pluto icon, or null. */
export const queryIconButton = (
  container: ParentNode,
  icon: string,
): HTMLButtonElement | null => queryIcon(container, icon)?.closest("button") ?? null;

/** Returns every button wrapping the given pluto icon, in document order. */
export const getIconButtons = (
  container: ParentNode,
  icon: string,
): HTMLButtonElement[] =>
  Array.from(container.querySelectorAll(`[aria-label="pluto-icon--${icon}"]`)).flatMap(
    (el) => {
      const button = el.closest("button");
      return button == null ? [] : [button];
    },
  );

/** Like queryIconButton, but throws when no button wraps the icon. */
export const getIconButton = (
  container: ParentNode,
  icon: string,
): HTMLButtonElement => {
  const button = queryIconButton(container, icon);
  if (button == null) throw new Error(`no button wrapping icon ${icon}`);
  return button;
};

/**
 * Finds the text input whose pluto node placeholder (a rendered sibling element, not
 * an HTML placeholder attribute) contains the given text.
 */
export const getInputByNodePlaceholder = (
  container: ParentNode,
  text: string,
): HTMLInputElement => {
  const placeholders = Array.from(
    container.querySelectorAll<HTMLElement>(".pluto-input__placeholder"),
  );
  const target = placeholders.find((el) => (el.textContent ?? "").includes(text));
  const input = target?.parentElement?.querySelector("input");
  if (input == null) throw new Error(`no input with node placeholder ${text}`);
  return input;
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

/** Finds the pluto input item element labeled with labelText in the document. */
export const getInputItem = (labelText: string): Element => {
  const item = screen.getByText(labelText).closest(".pluto-input__item");
  if (item == null) throw new Error(`no input item wraps label ${labelText}`);
  return item;
};

/**
 * Finds the input element of the pluto input item labeled with labelText. Pluto input
 * labels are sibling elements, so role/label queries cannot reach the input.
 */
export const getLabeledInput = (labelText: string): HTMLInputElement =>
  getBySelector<HTMLInputElement>(getInputItem(labelText), "input");

/**
 * Finds the dialog-select trigger of the pluto input item labeled with labelText. The
 * trigger renders no accessible role or stable text of its own.
 */
export const getLabeledDialogTrigger = (labelText: string): HTMLElement =>
  getBySelector<HTMLElement>(getInputItem(labelText), ".pluto-dialog__trigger");

/**
 * Reports whether a pluto element is rendered in its disabled state. Pluto marks
 * disabled buttons with a class rather than the disabled attribute.
 */
export const isPlutoDisabled = (el: Element): boolean =>
  el.classList.contains("pluto--disabled");

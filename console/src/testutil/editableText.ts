// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, waitFor } from "@testing-library/react";

const EDITABLE_CLASS = "pluto-text--editable";
const EDITING_SELECTOR = `.${EDITABLE_CLASS}[contenteditable="true"]`;

/**
 * Finds the editable text element with the given id — the target of a Text.edit
 * rename flow. Throws when no such element is rendered.
 */
export const findEditableText = (id: string): HTMLElement => {
  const el = document.querySelector<HTMLElement>(
    `#${CSS.escape(id)}.${EDITABLE_CLASS}`,
  );
  if (el == null) throw new Error(`editable text element ${id} not found`);
  return el;
};

/**
 * Waits until the editable text element with `id` has entered editing mode and
 * returns it.
 */
export const awaitTextEditing = async (id: string): Promise<HTMLElement> =>
  await waitFor(() => {
    const el = findEditableText(id);
    if (el.getAttribute("contenteditable") !== "true")
      throw new Error(`editable text element ${id} is not in editing mode`);
    return el;
  });

/**
 * Waits for any editable text to enter editing mode and returns the editable
 * element. Used when the edited element's id is generated inside the code under test.
 */
export const awaitTextEditingElement = async (): Promise<HTMLElement> =>
  await waitFor(() => {
    const el = document.querySelector<HTMLElement>(EDITING_SELECTOR);
    if (el == null) throw new Error("no text element is in editing mode");
    return el;
  });

/** Waits until no editable text is in editing mode. */
export const awaitTextEditingExit = async (): Promise<void> => {
  await waitFor(() => {
    if (document.querySelector(EDITING_SELECTOR) != null)
      throw new Error("a text element is still in editing mode");
  });
};

/**
 * Commits an in-progress text edit on `el` by replacing its content with `value` and
 * pressing Enter.
 */
export const commitTextEdit = (el: HTMLElement, value: string): void => {
  el.textContent = value;
  fireEvent.keyDown(el, { key: "Enter" });
};

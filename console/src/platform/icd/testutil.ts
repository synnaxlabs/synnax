// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { icd, type Synnax } from "@synnaxlabs/client";
import type { payload } from "@synnaxlabs/x";
import { fireEvent, screen, waitFor, within } from "@testing-library/react";
import { expect } from "vitest";
import { userEvent } from "vitest/browser";

import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { findDialogTriggerByText } from "@/testutil";

/**
 * Opens the message format field that shows current, searches for an ICD name, and
 * waits for the search to answer. Until then the list shows its first page, and the
 * answer replaces those rows, so a row found there can be detached by the next read.
 * @returns the dialog.
 */
export const searchMessageFormats = async (current: string, icdName: string) => {
  fireEvent.click(await findDialogTriggerByText(current));
  const dialogs = await screen.findAllByRole("dialog");
  const dialog = within(dialogs[dialogs.length - 1]);
  fireEvent.change(await dialog.findByPlaceholderText("Search message formats..."), {
    target: { value: icdName },
  });
  await waitFor(() => {
    const options = within(dialog.getByRole("listbox")).queryAllByRole("option");
    if (
      options.length === 0 ||
      options.some((o) => o.textContent?.includes(icdName) !== true)
    )
      throw new Error(`the search for ${icdName} has not answered`);
  });
  return dialog;
};

/** Picks the message next of the ICD named icdName in the field that shows current. */
export const pickMessageFormat = async (
  current: string,
  next: string,
  icdName: string,
): Promise<void> => {
  const dialog = await searchMessageFormats(current, icdName);
  const option = await waitFor(() => {
    const options = within(dialog.getByRole("listbox")).queryAllByRole("option");
    const match = options.find((o) => within(o).queryByText(next) != null);
    if (match == null) throw new Error(`no ${next} of ${icdName}`);
    return match;
  });
  fireEvent.click(option);
};

/**
 * Opens the message format field that shows current, and clicks its create item named
 * text, such as "New text message".
 */
export const pickNewMessage = async (current: string, text: string): Promise<void> => {
  fireEvent.click(await findDialogTriggerByText(current));
  const dialogs = await screen.findAllByRole("dialog");
  const dialog = within(dialogs[dialogs.length - 1]);
  fireEvent.click(await dialog.findByRole("option", { name: text }));
};

/** A message of an ICD. */
interface MessageRef {
  icd: icd.Key;
  message: icd.MessageKey;
}

const fieldsPath = (message: icd.MessageKey): string[] => [
  "messages",
  message,
  "payload",
  "fields",
];

/** Gives a field a new kind, in the ICD. after names the field it then follows. */
export const retype = async (
  client: Synnax,
  { icd: key, message }: MessageRef,
  field: icd.JSONField,
  kind: payload.Kind,
  after?: icd.FieldKey,
): Promise<void> => {
  const path = fieldsPath(message);
  await client.icds.dispatch(key, [
    icd.remove({ path, key: field.key }),
    icd.insert({ path, value: { ...field, type: kind }, after }),
  ]);
};

/** Replaces the field of a message that has the key of field, in the ICD. */
export const setField = async (
  client: Synnax,
  { icd: key, message }: MessageRef,
  field: icd.JSONField,
): Promise<void> => {
  await client.icds.dispatch(
    key,
    icd.set({ path: [...fieldsPath(message), field.key], value: field }),
  );
};

/** Removes a field of a message, in the ICD. */
export const removeField = async (
  client: Synnax,
  { icd: key, message }: MessageRef,
  field: icd.JSONField,
): Promise<void> => {
  await client.icds.dispatch(
    key,
    icd.remove({ path: fieldsPath(message), key: field.key }),
  );
};

/** @returns the messages that the editor of the ICD selects in the store. */
export const selectedIn = (
  store: { getState: () => Session.State },
  key: icd.Key,
): icd.MessageKey[] | undefined =>
  documentIn(store.getState()[Session.ICD.SLICE_NAME], key)?.selected;

/** @returns the field that the editor of the ICD focuses next, in the store. */
export const focusedIn = (
  store: { getState: () => Session.State },
  key: icd.Key,
): icd.FieldKey | undefined =>
  documentIn(store.getState()[Session.ICD.SLICE_NAME], key)?.focused;

/** @returns whether the store marks the ICD as just created, in the main window. */
export const createdIn = (
  store: { getState: () => Session.State },
  key: icd.Key,
): boolean | undefined =>
  documentIn(store.getState()[Session.ICD.SLICE_NAME], key)?.created;

/** @returns the title of the group of the field picker that note marks. */
export const titleOf = (note: HTMLElement): string =>
  note.parentElement?.firstChild?.textContent ?? "";

/** @returns whether the field picker fades value, the name of one of its values. */
export const isDimmed = (value: HTMLElement): boolean =>
  value.closest(".console--dimmed") != null;

/** Types text into the input with label, then commits it on blur. */
export const typeText = async (
  label: RegExp,
  text: string,
): Promise<HTMLInputElement> => {
  const input = await screen.findByLabelText<HTMLInputElement>(label);
  fireEvent.focus(input);
  fireEvent.change(input, { target: { value: text } });
  fireEvent.blur(input);
  return input;
};

/** Waits for input to refuse text, then checks that a hover shows error. */
export const expectRefused = async (
  input: HTMLInputElement,
  text: string,
  error: string,
): Promise<void> => {
  await waitFor(() => expect(input.getAttribute("aria-invalid")).toBe("true"));
  expect(input.value).toBe(text);
  await userEvent.hover(input);
  await screen.findByText(error);
};

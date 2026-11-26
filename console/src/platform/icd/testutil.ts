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

import { Session } from "@/session";
import { documentIn } from "@/session/window/testutil";
import { findDialogTriggerByText } from "@/testutil";

/**
 * Opens the message format field that shows current, and searches for an ICD name.
 * @returns the dialog.
 */
export const searchMessageFormats = async (current: string, icdName: string) => {
  fireEvent.click(await findDialogTriggerByText(current));
  const dialogs = await screen.findAllByRole("dialog");
  const dialog = within(dialogs[dialogs.length - 1]);
  fireEvent.change(await dialog.findByPlaceholderText("Search message formats..."), {
    target: { value: icdName },
  });
  return dialog;
};

/**
 * Picks the message next of the ICD named icdName in the message format field that
 * shows current, once the search answers. Until then the list shows its first page,
 * which unmounts a row out of view once the list measures its height, so a row found
 * there can be gone by the click.
 */
export const pickMessageFormat = async (
  current: string,
  next: string,
  icdName: string,
): Promise<void> => {
  const dialog = await searchMessageFormats(current, icdName);
  const option = await waitFor(() => {
    const options = within(dialog.getByRole("listbox")).queryAllByRole("option");
    if (options.some((o) => o.textContent?.includes(icdName) !== true))
      throw new Error(`the search for ${icdName} has not answered`);
    const match = options.find((o) => within(o).queryByText(next) != null);
    if (match == null) throw new Error(`no ${next} of ${icdName}`);
    return match;
  });
  fireEvent.click(option);
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

/** @returns the title of the group of the field picker that note marks. */
export const titleOf = (note: HTMLElement): string =>
  note.parentElement?.firstChild?.textContent ?? "";

/** @returns whether the field picker fades value, the name of one of its values. */
export const isDimmed = (value: HTMLElement): boolean =>
  value.closest(".console--dimmed") != null;

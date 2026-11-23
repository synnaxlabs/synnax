// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen, waitFor, within } from "@testing-library/react";

/**
 * Searches the select for the name, then clicks its option once the search answers.
 * Until then the list shows its first page, which unmounts a row out of view once the
 * list measures its height, so a row found there can be gone by the click. Options
 * pinned outside the list, such as a create option in a footer, do not count.
 *
 * @param search - The placeholder of the search field, such as "Search channels...".
 * @param scope - The element that holds the field and the list. Defaults to the dialog
 * of the only field on screen with the placeholder.
 */
export const pickOption = async (
  name: string,
  search: string,
  scope?: HTMLElement,
): Promise<void> => {
  const field = await (scope == null ? screen : within(scope)).findByPlaceholderText(
    search,
  );
  const holder = scope ?? field.closest<HTMLElement>('[role="dialog"]');
  if (holder == null) throw new Error(`no dialog holds the field "${search}"`);
  fireEvent.change(field, { target: { value: name } });
  const option = await waitFor(() => {
    const list = within(holder).getByRole("listbox");
    const [option, ...rest] = within(list).queryAllByRole("option");
    if (option?.textContent?.includes(name) !== true || rest.length > 0)
      throw new Error(`the search for ${name} has not answered`);
    return option;
  });
  fireEvent.click(option);
};

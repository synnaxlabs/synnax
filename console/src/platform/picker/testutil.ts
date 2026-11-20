// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, screen } from "@testing-library/react";
import { plural } from "pluralize";

/**
 * Searches the open picker of a resource, such as "secret", and returns the option of
 * the entry with the name. It never returns the create option, which names the search
 * too.
 */
export const findOption = async (
  resourceName: string,
  name: string,
  term: string = name,
): Promise<HTMLElement> => {
  const search = await screen.findByPlaceholderText(
    `Search ${plural(resourceName)}...`,
  );
  fireEvent.change(search, { target: { value: term } });
  return await screen.findByRole("option", {
    name: (shown) => shown.includes(name) && !shown.startsWith("Create "),
  });
};

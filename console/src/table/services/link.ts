// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Link } from "@/link";
import { type State } from "@/table/slice";
import { create } from "@/table/Table";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  place,
  handleException,
}): Promise<boolean> => {
  if (resource !== "table") return false;
  try {
    const table = await client.workspaces.table.retrieve(resourceKey);
    const layoutCreator = create({
      ...(table.data as unknown as State),
      key: table.key,
      name: table.name,
    });
    place(layoutCreator);
  } catch (e) {
    handleException(e, "Failed to open table from URL");
  }
  return true;
};

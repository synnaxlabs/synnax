// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Link } from "@/link";
import { create } from "@/log/Log";
import { type State } from "@/log/slice";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  client,
  placer,
  handleException,
}): Promise<boolean> => {
  if (resource !== "log") return false;
  try {
    const log = await client.workspaces.log.retrieve(resourceKey);
    const layoutCreator = create({
      ...(log.data as unknown as State),
      key: log.key,
      name: log.name,
    });
    placer(layoutCreator);
  } catch (e) {
    handleException(e, "Failed to open log from URL");
  }
  return true;
};

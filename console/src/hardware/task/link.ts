// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";

import { retrieveAndPlaceLayout } from "@/hardware/task/ontology";
import { type Link } from "@/link";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  placer,
  client,
  handleException,
}): Promise<boolean> => {
  if (resource !== task.ONTOLOGY_TYPE) return false;
  try {
    await retrieveAndPlaceLayout(client, resourceKey, placer);
  } catch (e) {
    handleException(e, "Failed to open task from URL");
  }
  return true;
};

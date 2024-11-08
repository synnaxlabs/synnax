// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { task } from "@synnaxlabs/client";

import { createTaskLayout } from "@/hardware/task/ontology";
import { Link } from "@/link";

export const linkHandler: Link.Handler = async ({
  resource,
  resourceKey,
  placer,
  client,
  addStatus,
}): Promise<boolean> => {
  if (resource !== task.ONTOLOGY_TYPE) return false;
  try {
    const task_ = await client.hardware.tasks.retrieve(resourceKey);
    const layout = createTaskLayout(resourceKey, task_.type);
    placer(layout);
  } catch (e) {
    if (!(e instanceof Error)) throw e;
    addStatus({
      variant: "error",
      description: "Could not load task from URL",
      message: e.message,
    });
  }
  return true;
};

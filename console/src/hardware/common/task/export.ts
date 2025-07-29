// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { DisconnectedError } from "@synnaxlabs/client";

import { Export } from "@/export";

export const extract: Export.Extractor = async (key, { client }) => {
  if (client == null) throw new DisconnectedError();
  const task = await client.hardware.tasks.retrieve({ key });
  return { data: JSON.stringify(task.config), name: task.name };
};

export const useExport = () => Export.use(extract, "task");

// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Link } from "@/link";
import { Log } from "@/log";

export const handleLink: Link.Handler = async ({ client, key, placeLayout }) => {
  const log = await client.workspaces.logs.retrieve({ key });
  placeLayout(Log.create({ ...log.data, ...log }));
};

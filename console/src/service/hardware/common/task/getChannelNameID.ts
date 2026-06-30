// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type WriteChannelType } from "@/hardware/common/task/types";

export const getChannelNameID = (key: string, type?: WriteChannelType) => {
  switch (type) {
    case "cmd":
      return `text-cmd-${key}`;
    case "state":
      return `text-state-${key}`;
    default:
      return `text-${key}`;
  }
};

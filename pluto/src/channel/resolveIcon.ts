// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { telem } from "@synnaxlabs/x/telem";
import { Icon } from "@synnaxlabs/lyra/icon";
import { channel } from "@synnaxlabs/client";

import { Telem } from "@/telem";

export const resolveIcon = (ch?: channel.Payload): Icon.FC => {
  if (ch == null) return Icon.Channel;
  if (channel.isCalculated(ch)) return Icon.Calculation;
  return Telem.resolveDataTypeIcon(new telem.DataType(ch.dataType)) ?? Icon.Channel;
};

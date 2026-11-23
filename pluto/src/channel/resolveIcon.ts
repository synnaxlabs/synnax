// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { channel } from "@synnaxlabs/client";
import { Icon } from "@synnaxlabs/lyra/icon";
import { DataType } from "@synnaxlabs/x";

import { Telem } from "@/telem";

/**
 * @returns the icon of the channel: a calculation icon when a calculation writes it,
 * and otherwise the icon of its data type.
 */
export const resolveIcon = (
  ch?: Pick<channel.Payload, "dataType">,
  calculated: boolean = false,
): Icon.FC => {
  if (ch == null) return Icon.Channel;
  if (calculated) return Icon.Calculation;
  return Telem.resolveDataTypeIcon(new DataType(ch.dataType)) ?? Icon.Channel;
};

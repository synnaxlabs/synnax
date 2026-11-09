// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@/context";
import { type Status } from "@/status";

/** What an item tells the control inside it. */
export interface ItemContextValue {
  /** The id of the item's visible label. */
  labelID?: string;
  status?: Status.Variant;
  message?: string;
  /** The id of the help text that shows the message. */
  messageID?: string;
  /** Whether the item has no help text, so its control shows the message. */
  helpTextHidden: boolean;
}

/** Set by an item for its control. Null outside an item and inside an input. */
export const [ItemContext, useItemContext] = context.create<ItemContextValue | null>({
  defaultValue: null,
  displayName: "Input.ItemContext",
});

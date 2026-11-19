// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type channel, NotFoundError } from "@synnaxlabs/client";
import { Text } from "@synnaxlabs/lyra/text";
import { primitive } from "@synnaxlabs/x";
import { type ReactElement } from "react";

import { useResult } from "@/channel/queries";
import { Flux } from "@/flux";
import { telem } from "@/telem/aether";

/** The text that a view shows in place of a channel that the Core deleted. */
export const DELETED_TEXT = telem.DELETED_STATUS.message;

/**
 * @returns true once the Core holds no channel with key, as after its delete. The zero
 * key is never deleted. A channel that the user cannot read is not deleted.
 */
export const useDeleted = (key: channel.Key): boolean => {
  const res = useResult(primitive.isZero(key) ? null : { key });
  if (res.variant !== "error") return false;
  const { error } = res.status.details;
  return NotFoundError.matches(error.cause) || Flux.DeletedError.matches(error);
};

/** Says that the Core deleted the channel, in a view of its value. */
export const Deleted = (): ReactElement => (
  <Text.Text level="small" color={9} overflow="fade">
    {DELETED_TEXT}
  </Text.Text>
);

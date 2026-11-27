// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import {
  type channel,
  NotFoundError,
  query,
  type Synnax as Client,
} from "@synnaxlabs/client";
import { useMemoDeepEqual } from "@synnaxlabs/lyra/memo";
import { Text } from "@synnaxlabs/lyra/text";
import { type destructor, primitive } from "@synnaxlabs/x";
import { type ReactElement, useEffect, useState } from "react";

import { retrieveDefinition } from "@/channel/aether/queries";
import { useResult } from "@/channel/queries";
import { Flux } from "@/flux";
import { Synnax } from "@/synnax";
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

// Calls onChange with whether the Core holds no channel with key: once it knows, then
// on each change. A failure other than a missing channel calls nothing.
const watchDeleted = (
  client: Client,
  key: channel.Key,
  onChange: (deleted: boolean) => void,
): destructor.Destructor => {
  const params = { client, query: { key } };
  let pushed = false;
  let closed = false;
  const disconnect = retrieveDefinition.onChange?.(params, (result) => {
    // An undefined push is an invalidation that the cache repairs.
    if (result === undefined || closed) return;
    pushed = true;
    onChange(query.Deleted.matches(result));
  });
  const cached = retrieveDefinition.getCached?.(params);
  if (cached !== undefined) onChange(query.Deleted.matches(cached));
  else
    retrieveDefinition.retrieve(params).then(
      () => {
        if (!closed && !pushed) onChange(false);
      },
      (e: unknown) => {
        if (!closed && !pushed && NotFoundError.matches(e)) onChange(true);
      },
    );
  return () => {
    closed = true;
    disconnect?.();
  };
};

const NONE: ReadonlySet<channel.Key> = new Set();

/**
 * @returns true once the Core holds no channel with one of keys, as useDeleted reads
 * each. keys can change in number.
 */
export const useAnyDeleted = (keys: channel.Key[]): boolean => {
  const client = Synnax.use();
  const watched = useMemoDeepEqual(
    [...new Set(keys.filter(primitive.isNonZero))].sort((a, b) => a - b),
  );
  const [deleted, setDeleted] = useState(NONE);
  useEffect(() => {
    if (client == null) return;
    const stops = watched.map((key) =>
      watchDeleted(client, key, (gone) =>
        setDeleted((prev) => {
          if (prev.has(key) === gone) return prev;
          const next = new Set(prev);
          if (gone) next.add(key);
          else next.delete(key);
          return next;
        }),
      ),
    );
    return () => stops.forEach((stop) => stop());
  }, [client, watched]);
  return watched.some((key) => deleted.has(key));
};

export interface DeletedProps {
  /** color tints the text, as a stale value's color does. */
  color?: Text.TextProps["color"];
}

/** Says that the Core deleted the channel, in a view of its value. */
export const Deleted = ({ color = 9 }: DeletedProps): ReactElement => (
  <Text.Text level="small" color={color} overflow="fade">
    {DELETED_TEXT}
  </Text.Text>
);

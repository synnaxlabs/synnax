// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";

import { encodes } from "@/platform/icd/dataType";

const channelBinding = (field: icd.FieldKey): icd.ChannelWriteBinding => ({
  type: "channel",
  field,
  channel: 0,
});

/**
 * @returns bindings, changed to suit fields: no binding of a field that is gone, no
 * fixed value of another kind, and no data type that its field does not take. Each
 * field with no default and no binding takes a channel. When no field takes one, the
 * first field does, because the Core refuses a message that binds no channels.
 */
export const suit = (
  bindings: icd.WriteBinding[],
  fields: icd.JSONField[],
): icd.WriteBinding[] => {
  const byKey = new Map(fields.map((f) => [f.key, f]));
  const kept = bindings.flatMap((b): icd.WriteBinding[] => {
    const field = byKey.get(b.field);
    if (field == null) return [];
    if (b.type === "value") return b.value.type === field.type ? [b] : [];
    if (b.dataType == null || encodes(field, b.dataType)) return [b];
    return [{ type: "channel", field: b.field, channel: b.channel }];
  });
  const bound = new Set(kept.map(({ field }) => field));
  const suited = [
    ...kept,
    ...fields
      .filter((f) => f.default == null && !bound.has(f.key))
      .map((f) => channelBinding(f.key)),
  ];
  const first = fields.at(0);
  if (first == null || suited.some(({ type }) => type === "channel")) return suited;
  return [channelBinding(first.key), ...suited.filter((b) => b.field !== first.key)];
};

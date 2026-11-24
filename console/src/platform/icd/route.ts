// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import type { icd } from "@synnaxlabs/client";

/**
 * @returns the message of messages that the Core gives a document with fields: the one
 * whose present rule is at one of the fields or holds one, else the one with no rule.
 */
export const route = <M extends { match?: icd.JSONMatch }>(
  fields: { pointer: string }[],
  messages: M[],
): M | undefined => {
  const holds = (at: string): boolean =>
    fields.some(({ pointer }) => pointer === at || pointer.startsWith(`${at}/`));
  return (
    messages.find(({ match }) => match?.type === "present" && holds(match.pointer)) ??
    messages.find(({ match }) => match == null)
  );
};

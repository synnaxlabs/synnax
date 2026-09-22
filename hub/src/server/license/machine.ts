// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { badRequest } from "@/server/errors";

/** MAX_NAME_LENGTH bounds the name a machine carries in the ledger. */
export const MAX_NAME_LENGTH = 64;

/**
 * readName trims a posted machine name to its bound.
 * @throws {HTTPError} 400 when nothing is left of it.
 */
export const readName = (raw: string | undefined): string => {
  const name = (raw ?? "").trim().slice(0, MAX_NAME_LENGTH);
  if (name === "") throw badRequest("The machine needs a name");
  return name;
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { errors } from "@synnaxlabs/x";

export class FluxError extends errors.createTyped("flux") {}

const Base = FluxError.sub("deleted");

const nameOf = (corpse: unknown): string | undefined => {
  if (typeof corpse !== "object" || corpse === null || !("name" in corpse))
    return undefined;
  const { name } = corpse;
  return typeof name === "string" ? name : undefined;
};

/**
 * Thrown to the error boundary when a query's cached answer is a deletion. The
 * corpse itself stays in the client's cache, typed; only the deleted record's
 * name travels with the error, for display.
 */
export class DeletedError extends Base {
  /** The deleted record's name, absent when the corpse carried none. */
  readonly corpseName?: string;

  constructor(message: string, corpse: unknown) {
    super(message);
    this.corpseName = nameOf(corpse);
  }

  static override readonly matches = (e: unknown): e is DeletedError => Base.matches(e);
}

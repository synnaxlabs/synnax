// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { ValidationError } from "@/errors";

const RULE =
  "must start with a letter or underscore and contain only letters, digits, and underscores";

/** identifierZ checks the name of a module or a directory, which Arcs import by. */
export const identifierZ = z.string().regex(/^[a-zA-Z_][a-zA-Z0-9_]*$/, `Name ${RULE}`);

/** @throws {ValidationError} when name is not an identifier. */
export const validateIdentifier = (name: string): void => {
  if (!identifierZ.safeParse(name).success)
    throw new ValidationError(`Name "${name}" ${RULE}`);
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";
import { type z } from "zod";

import { useContext } from "@/form/Context";

/**
 * Checks the value at the path with the schema while the caller is mounted, as
 * {@link ContextValue.addCheck} does. A null schema checks nothing, and a new schema
 * replaces the last one.
 */
export const useCheck = (path: string, schema: z.ZodType | null): void => {
  const { addCheck } = useContext(undefined, `useCheck(${path})`);
  useEffect(
    () => (schema == null ? undefined : addCheck(path, schema)),
    [addCheck, path, schema],
  );
};

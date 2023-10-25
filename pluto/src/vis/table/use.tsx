// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { Aether } from "@/aether";
import { table } from "@/vis/table/aether";

export interface UseProps extends z.input<typeof table.tableStateZ> {
  aetherKey: string;
}

export interface SourceProps extends z.input<typeof table.tableSourceZ> {}

export const use = ({
  aetherKey,
  ...props
}: UseProps): Aether.UseReturn<typeof table.tableStateZ> =>
  Aether.use({
    aetherKey,
    type: table.Table.TYPE,
    schema: table.tableStateZ,
    initialState: props,
  });

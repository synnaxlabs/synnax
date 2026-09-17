// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import pre from "@/components/code/Block.astro";
import Details from "@/components/details/Details.astro";
import table from "@/components/table/Table.astro";
import { textFactory } from "@/components/text/Text";

export const mdxOverrides = {
  pre,
  table,
  h1: textFactory({ level: "h1", includeAnchor: true }),
  h2: textFactory({ level: "h2", includeAnchor: true }),
  h3: textFactory({ level: "h3", includeAnchor: true }),
  details: Details,
};

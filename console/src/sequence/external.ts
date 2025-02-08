// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Layout } from "@/layout";
import { Configure, SEQUENCE_SELECTABLE } from "@/sequence/Configure";
import { TYPE } from "@/sequence/types";

export * from "@/sequence/types";

export const LAYOUTS: Record<string, Layout.Renderer> = {
  [TYPE]: Configure,
};

export const SELECTABLES: Layout.Selectable[] = [SEQUENCE_SELECTABLE];

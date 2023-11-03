// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { ValueSpec } from "@/vis/pid/element/Value";
import { type Spec } from "@/vis/table/element/element";
import { LabelSpec } from "@/vis/table/element/label";

export const REGISTRY: Record<string, Spec<any>> = {
  [ValueSpec.type]: ValueSpec,
  [LabelSpec.type]: LabelSpec,
};

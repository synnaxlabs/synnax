// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type z } from "zod";

import { Aether } from "@/aether";
import { range } from "@/lineplot/range/aether";

interface AnnotationProps
  extends z.input<typeof range.annotationStateZ>,
    Aether.ComponentProps {}

export const Annotation = ({ aetherKey, ...rest }: AnnotationProps): null => {
  Aether.use({
    aetherKey,
    type: range.Annotation.TYPE,
    schema: range.annotationStateZ,
    initialState: rest,
  });
  return null;
};

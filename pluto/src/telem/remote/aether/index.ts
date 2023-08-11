// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

import { Factory } from "@/telem/remote/aether/factory";
import { numericProps } from "@/telem/remote/aether/numeric";
import { dynamicXYProps, xyProps } from "@/telem/remote/aether/xy";

export const AetherRemoteTelem = {
  Factory,
  props: {
    numeric: numericProps as z.ZodTypeAny,
    xy: xyProps as z.ZodTypeAny,
    dynamicXY: dynamicXYProps as z.ZodTypeAny,
  },
};

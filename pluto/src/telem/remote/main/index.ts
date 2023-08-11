// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { NumericProps as RemoteTelemNumericProps } from "@/telem/remote/aether/numeric";
import {
  XYProps as RemoteTelemXYProps,
  DynamicXYProps as RemoteTelemDynamicXyProps,
} from "@/telem/remote/aether/xy";
import {
  NumericForm,
  NumericFormProps as RemoteTelemNumericFormProps,
} from "@/telem/remote/main/forms";
import { useXY, useDynamicXY, useNumeric } from "@/telem/remote/main/hooks";

export const RemoteTelem = {
  useXY,
  useDynamicXY,
  useNumeric,
  Form: {
    Numeric: NumericForm,
  },
};

export type {
  RemoteTelemNumericProps,
  RemoteTelemXYProps,
  RemoteTelemDynamicXyProps,
  RemoteTelemNumericFormProps,
};

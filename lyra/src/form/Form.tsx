// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactElement } from "react";
import { type z } from "zod";

import { Context, type ContextValue } from "@/form/Context";

export const Form = <Z extends z.ZodType>({
  children,
  ...rest
}: PropsWithChildren<ContextValue<Z>>): ReactElement => (
  <Context value={rest as ContextValue}>{children}</Context>
);

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form } from "@synnaxlabs/lyra/form";
import { type z } from "zod";

/**
 * Whether the surrounding task form renders read-only because the subject holds no
 * update grant on the task.
 */
export const useIsPreview = <Schema extends z.ZodType>(
  ctx?: Form.ContextValue<Schema>,
) => Form.useContext<Schema>(ctx).mode === "preview";

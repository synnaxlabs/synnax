// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { z } from "zod";

export const nullableArrayZ = <Z extends z.ZodTypeAny>(item: Z) =>
  z.union([z.null().transform(() => [] as z.output<Z>[]), item.array()]);

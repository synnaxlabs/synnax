// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

<<<<<<<< HEAD:pluto/src/status/core/CreateIcon.ts
import { Icon } from "@/icon";

export const CreateIcon = Icon.createComposite(Icon.Status, { topRight: Icon.Add });
========
import z from "zod";

export const nullableZ = <Z extends z.ZodType>(item: Z) =>
  z.union([
    z.union([z.null(), z.undefined()]).transform<z.infer<Z>[]>(() => []),
    item.array(),
  ]);
>>>>>>>> 46448fc2c2846ab4b42a73fe24427827c9eec1a1:x/ts/src/array/nullable.ts

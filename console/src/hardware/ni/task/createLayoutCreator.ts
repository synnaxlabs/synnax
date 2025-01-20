// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type task } from "@synnaxlabs/client";

import { Common } from "@/hardware/common";

export const createLayoutCreator = <P extends task.Payload>(
  type: string,
  defaultName: string = "New NI Task",
) => Common.Task.createLayoutCreator<P>(type, defaultName, "Logo.NI");

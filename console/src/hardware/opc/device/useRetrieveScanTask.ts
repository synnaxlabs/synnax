// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type rack } from "@synnaxlabs/client";
import { Task } from "@synnaxlabs/pluto";

import {
  SCAN_SCHEMAS,
  SCAN_TYPE,
  type scanConfigZ,
  type scanStatusDataZ,
  type scanTypeZ,
} from "@/hardware/opc/task/types";

const { useRetrieve } = Task.createRetrieve<
  typeof scanTypeZ,
  typeof scanConfigZ,
  typeof scanStatusDataZ
>({ schemas: SCAN_SCHEMAS });

export const useRetrieveScanTask = (rack: rack.Key) =>
  useRetrieve({ type: SCAN_TYPE, rack }).data;

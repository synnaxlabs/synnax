// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useCallback } from "react";

export const useListenForChanges = () => {
  const addStatus = Status.useAdder();
  const handleSet = useCallback(addStatus, [addStatus]);
  Status.useSetSynchronizer(handleSet);
};

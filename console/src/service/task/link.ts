// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { retrieveAndPlaceLayout } from "@/hardware/task/layouts";
import { type Link } from "@/service/link";
import { Layout } from "@/layout";

export const useLink = (): Link.Handler => {
  const placeLayout = Layout.usePlacer();
  return useCallback(
    async ({ client, key }) => await retrieveAndPlaceLayout(client, key, placeLayout),
    [placeLayout],
  );
};

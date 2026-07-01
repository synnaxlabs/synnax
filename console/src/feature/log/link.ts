// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback } from "react";

import { Layout } from "@/platform/layout";
import { type Link } from "@/platform/link";
import { create } from "@/platform/log/layout";

export const useLink = (): Link.Handler => {
  const placeLayout = Layout.usePlacer();
  return useCallback(
    async ({ client, key }) => {
      const { name } = await client.logs.retrieve({ key });
      placeLayout(create({ key, name }));
    },
    [placeLayout],
  );
};

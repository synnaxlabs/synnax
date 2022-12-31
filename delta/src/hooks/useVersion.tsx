// Copyright 2022 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useState } from "react";

import { getVersion } from "@tauri-apps/api/app";

import { useAsyncEffect } from "./useAsyncEffect";

/**
 * useVersion gets the application version.
 * @returns The application version in the format "vX.X.X".
 */
export const useVersion = (): string => {
  const [v, setV] = useState<string>("");
  useAsyncEffect(async () => setV("v" + (await getVersion())), []);
  return v;
};

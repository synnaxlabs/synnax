// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect } from "@synnaxlabs/pluto";
import { useDispatch } from "react-redux";
import { set } from "@/version/slice";

interface ElectronAPI {
  get: () => Promise<string>;
}
const ELECTRON_API_KEY = "versionAPI";

/** @returns the tauri application version as exposed by the tauri apps API. */
export const electronVersion = async (): Promise<string> => {
  if (!(ELECTRON_API_KEY in window)) throw new Error("Electron Version API not found.");
  return await (window as { [ELECTRON_API_KEY]: ElectronAPI })[ELECTRON_API_KEY].get();
};

export const useLoadElectron = (): void => {
  const d = useDispatch();
  useAsyncEffect(async () => {
    d(set(await electronVersion()));
  }, []);
};

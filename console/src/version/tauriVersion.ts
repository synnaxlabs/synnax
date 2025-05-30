// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useAsyncEffect } from "@synnaxlabs/pluto";
import { getVersion } from "@tauri-apps/api/app";
import { useDispatch } from "react-redux";

import { RUNTIME } from "@/runtime";
import { set } from "@/version/slice";

const tauriVersion = async (): Promise<string> => await getVersion();

export const useLoadTauri = (): void => {
  const dispatch = useDispatch();
  useAsyncEffect(async () => {
    if (RUNTIME !== "tauri") return;
    dispatch(set(await tauriVersion()));
  }, []);
};

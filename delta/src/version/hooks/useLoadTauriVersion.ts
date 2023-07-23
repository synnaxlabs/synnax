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

import { setVersion } from "../store";
import { tauriVersion } from "../tauriVersion";

export const useLoadTauriVersion = (): void => {
  const d = useDispatch();
  useAsyncEffect(async () => {
    d(setVersion(await tauriVersion()));
  }, []);
};

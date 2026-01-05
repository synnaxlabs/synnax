// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Synnax } from "@synnaxlabs/pluto";
import { useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { setActive } from "@/workspace/slice";

// useCheckCore is a hook run to make sure that the active workspace is not saved when
// switching clusters, making sure the user doesn't accidentally lose data because of a
// mismatch with a workspace that does not exist.
export const useCheckCore = (): void => {
  const currentClientKey = Synnax.use()?.key;
  const dispatch = useDispatch();
  const clientKeyRef = useRef(currentClientKey);
  useEffect(() => {
    if (clientKeyRef.current === currentClientKey) return;
    clientKeyRef.current = currentClientKey;
    dispatch(setActive(null));
  }, [currentClientKey, dispatch]);
};

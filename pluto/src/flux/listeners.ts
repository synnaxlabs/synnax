// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type MultiSeries } from "@synnaxlabs/x";
import { useEffect } from "react";

import { flux } from "@/flux/aether";
import { useAddListener } from "@/flux/Context";
import { Status } from "@/status";

export const useListener = (
  listeners: flux.ListenerSpec<MultiSeries, {}> | flux.ListenerSpec<MultiSeries, {}>[],
): void => {
  const addListener = useAddListener();
  const handleError = Status.useErrorHandler();
  useEffect(
    () => flux.mountListeners(addListener, handleError, listeners),
    [addListener, handleError, listeners],
  );
};

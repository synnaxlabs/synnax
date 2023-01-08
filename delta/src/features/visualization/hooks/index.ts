/*
 * Copyright 2023 Synnax Labs, Inc.
 *
 * Use of this software is governed by the Business Source License included in the file
 * licenses/BSL.txt.
 *
 * As of the Change Date specified in that file, in accordance with the Business Source
 * License, use of this software will be governed by the Apache License, Version 2.0,
 * included in the file licenses/APL.txt.
 */

import { useCallback } from "react";

import { useDispatch } from "react-redux";

import { updateVisualization as uv } from "../store";
import { Visualization } from "../types";

export const useUpdateVisualization = (
  key: string
): (<V extends Visualization>(v: V) => void) => {
  const d = useDispatch();
  return useCallback(
    <V extends Visualization>(v: V): void => {
      d(uv({ ...v, key }));
    },
    [d, key]
  );
};

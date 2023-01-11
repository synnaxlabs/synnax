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

import { DeepPartial } from "react-hook-form";
import { useDispatch } from "react-redux";

import { updateVis, useSelectSVis } from "../store";
import { ControlledVisProps, Vis } from "../types";

export const useControlledVis = <V extends Vis, SV extends Vis = V>(
  key?: string
): ControlledVisProps<V, SV> | undefined => {
  const dispatch = useDispatch();
  const sv = useSelectSVis<SV>(key);
  const onChange = useCallback(
    (vis: DeepPartial<V>) => dispatch(updateVis(vis as V)),
    [dispatch]
  );
  if (sv == null) return undefined;
  return {
    vis: sv,
    setVis: onChange,
  };
};

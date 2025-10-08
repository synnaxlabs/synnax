// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { usePrevious } from "@synnaxlabs/pluto";
import { primitive } from "@synnaxlabs/x";
import { useEffect } from "react";
import { useDispatch } from "react-redux";

import { useSelect } from "@/layout/selectors";
import { rename } from "@/layout/slice";

export const useSyncName = (
  layoutKey: string,
  externalName: string,
  onChange: (name: string) => void,
) => {
  const layoutName = useSelect(layoutKey)?.name;
  const prevLayoutName = usePrevious(layoutName);
  const dispatch = useDispatch();
  useEffect(() => {
    if (prevLayoutName == layoutName || prevLayoutName == null || layoutName == null)
      return;
    onChange(layoutName);
  }, [layoutName, onChange, prevLayoutName]);
  useEffect(() => {
    if (primitive.isNonZero(externalName))
      dispatch(rename({ key: layoutKey, name: externalName }));
  }, [externalName]);
};

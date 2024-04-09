// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { v4 as uuidv4 } from "uuid";

import { type LayoutState } from "@/layout/layout";

export interface Spec {
  variant: string;
  key: string;
}

export const create = (
  props: Omit<Partial<LayoutState>, "type">,
): Omit<LayoutState, "windowKey"> => {
  const {
    location = "mosaic",
    name = "Visualization",
    key = uuidv4(),
    window,
    tab,
  } = props;
  return {
    type: "vis",
    location,
    name,
    key,
    window,
    tab,
  };
};

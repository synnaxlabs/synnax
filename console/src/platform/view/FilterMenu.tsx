// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type PropsWithChildren, type ReactNode } from "react";

import { Search } from "@/platform/search";
import { useContext } from "@/platform/view/context";

export interface FilterMenuProps extends PropsWithChildren {}

export const FilterMenu = ({ children }: FilterMenuProps): ReactNode => {
  const { editable } = useContext("View.FilterMenu");
  if (!editable) return null;
  return <Search.FilterMenu>{children}</Search.FilterMenu>;
};

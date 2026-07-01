// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useContext as useCtx } from "@/primitive/view/context";

export const useContext = () => useCtx("View.useContext");

export * from "@/primitive/view/FilterMenu";
export * from "@/primitive/view/Form";
export * from "@/primitive/view/Items";
export * from "@/primitive/view/Search";
export * from "@/primitive/view/Toolbar";

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useContext as useCtx } from "@/platform/view/context";

export * from "@/platform/view/FilterChip";
export * from "@/platform/view/FilterMenu";
export * from "@/platform/view/Form";
export * from "@/platform/view/Frame";
export * from "@/platform/view/Items";
export * from "@/platform/view/Search";
export * from "@/platform/view/Toolbar";

export const useContext = () => useCtx("View.useContext");

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type Dispatch, type UnknownAction } from "@reduxjs/toolkit";
import { context } from "@synnaxlabs/pluto";

export interface ArcEditorContext {
  layoutKey: string;
  dispatch: Dispatch<UnknownAction>;
}

// This is a temporary type that is used for drilling the arc program key and
// the UNDOABLE dispatch down into the arc stage renderer. This function will
// be removed when arc no longer gets stored in redux.
export const [Provider, useArcEditorContext] = context.create<ArcEditorContext>({
  providerName: "ArcEditor.Provider",
  displayName: "ArcEditor.Context",
});

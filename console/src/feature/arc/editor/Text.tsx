// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Arc } from "@synnaxlabs/pluto";

import { TaskControls } from "@/feature/arc/editor/TaskControls";
import { Session } from "@/session";

export const Text = () => {
  // Background tabs stay mounted, so an ungated autofocus would let any of them pull
  // the cursor out of the tab the user is actually looking at.
  const getTabIsFocused = Session.Panel.useGetTabIsFocused();
  return (
    <>
      <Arc.Text.Editor autoFocus={getTabIsFocused()} />
      <TaskControls />
    </>
  );
};

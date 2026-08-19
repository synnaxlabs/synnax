// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent, getByLabelText } from "@testing-library/react";

/** Hovers the rendered ConnectionBadge's trigger to open its tooltip. */
export const hoverConnectionBadge = (container: HTMLElement): void => {
  fireEvent.pointerOver(getByLabelText(container, "Connection status"), {
    pointerType: "mouse",
  });
};

/** Clicks the rendered ConnectionBadge's trigger to open its dialog. */
export const clickConnectionBadge = (container: HTMLElement): void => {
  fireEvent.click(getByLabelText(container, "Connection status"));
};

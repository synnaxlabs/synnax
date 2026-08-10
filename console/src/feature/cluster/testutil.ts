// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { fireEvent } from "@testing-library/react";

import { getBySelector } from "@/testutil";

/**
 * Hovers the rendered ConnectionBadge's trigger to open its tooltip. The badge
 * renders an icon-only button with no accessible handle, so the structural selector
 * lives here. The tooltip opens on pointerover and ignores touch pointers.
 */
export const hoverConnectionBadge = (container: HTMLElement): void => {
  fireEvent.pointerOver(getBySelector(container, "button"), { pointerType: "mouse" });
};

/** Clicks the rendered ConnectionBadge's trigger to open its dialog. */
export const clickConnectionBadge = (container: HTMLElement): void => {
  fireEvent.click(getBySelector(container, "button"));
};

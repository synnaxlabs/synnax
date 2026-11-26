// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";

const SELECTOR = `button.${CSS.B("symbol-primitive-toggle")}`;

/** @returns the toggle button of the symbol in container, or null when it has none. */
export const queryButton = (container: ParentNode): HTMLButtonElement | null =>
  container.querySelector(SELECTOR);

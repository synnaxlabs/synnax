// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type KeyboardEvent } from "react";

const ACTIVATION_KEYS = [" ", "Enter"];

/**
 * Cancels the browser's synthetic click for Space and Enter so a focused schematic
 * control never actuates from the keyboard. Attach to both keydown and keyup: Enter
 * clicks on keydown and Space on keyup.
 */
export const blockActivation = (e: KeyboardEvent<HTMLElement>): void => {
  if (ACTIVATION_KEYS.includes(e.key)) e.preventDefault();
};

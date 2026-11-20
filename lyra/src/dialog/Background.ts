// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { CSS } from "@/css";

/** The class of the native modal dialog around a modal surface. */
export const BACKGROUND_CLASS = CSS.BE("dialog", "bg");

/** Whether the element sits in the modal on top, the last one opened. */
export const inTopModal = (el: Element): boolean => {
  const backgrounds = document.getElementsByClassName(BACKGROUND_CLASS);
  return backgrounds[backgrounds.length - 1]?.contains(el) === true;
};

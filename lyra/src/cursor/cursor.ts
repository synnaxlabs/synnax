// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties } from "react";

const EL_ID = "cursor-style";

export const setGlobalStyle = (
  cursor: CSSProperties["cursor"],
  el: HTMLElement = document.head,
): void => {
  clearGlobalStyle();
  const cursorStyle = document.createElement("style");
  cursorStyle.innerHTML = `*{cursor: ${cursor as string} !important;}`;
  cursorStyle.id = EL_ID;
  el.appendChild(cursorStyle);
};

export const clearGlobalStyle = (): void => {
  const cursorStyle = document.getElementById(EL_ID);
  if (cursorStyle != null) cursorStyle.remove();
};

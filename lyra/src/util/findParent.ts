// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

export const findParent = (
  el: HTMLElement | null,
  cb: (el: HTMLElement | null) => boolean,
): HTMLElement | null => {
  while (el != null && !cb(el)) el = el.parentElement;
  return el;
};

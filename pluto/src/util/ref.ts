// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Ref } from "react";

export const triggerRef = <T>(ref: Ref<T> | undefined, v: T): void => {
  if (typeof ref === "function") ref(v);
  // @ts-expect-error
  else if (ref != null) ref.current = v;
};

export const mergeRefs =
  <T>(...refs: Array<Ref<T> | undefined>) =>
  (e: T | null) => {
    refs.forEach((ref) => triggerRef(ref, e));
  };

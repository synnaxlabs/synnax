// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Form as PForm } from "@synnaxlabs/lyra/form";
import { useSyncExternalStore } from "react";

import type { Formats } from "@/platform/icd/Formats";

/**
 * @returns the plain data that read gives from the enclosing form and the open formats.
 * Each change to either reads it again, and the caller renders again only when its JSON
 * changes.
 */
export const useStructure = <S>(formats: Formats, read: () => S): S => {
  const { bind } = PForm.useContext();
  const subscribe = (onChange: () => void) => {
    const unbinds = [
      bind(onChange),
      ...[...formats.values()].map((f) => f.bind(onChange)),
    ];
    return () => unbinds.forEach((unbind) => unbind());
  };
  const getSnapshot = () => JSON.stringify(read());
  const snapshot = useSyncExternalStore(subscribe, getSnapshot, getSnapshot);
  return JSON.parse(snapshot) as S;
};

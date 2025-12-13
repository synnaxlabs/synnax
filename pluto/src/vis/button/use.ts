// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { button } from "@/vis/button/aether";

export type Mode = button.Mode;
export const MODES = button.MODES;

export interface UseProps extends z.input<typeof button.buttonStateZ> {
  aetherKey: string;
}

export interface UseReturn {
  onClick: () => void;
  onMouseDown: () => void;
  onMouseUp: () => void;
}

export const use = ({ aetherKey, sink, mode }: UseProps): UseReturn => {
  const {
    methods: { onMouseDown, onMouseUp },
  } = Aether.useUnidirectional({
    aetherKey,
    type: button.Button.TYPE,
    schema: button.buttonStateZ,
    methods: button.buttonMethodsZ,
    state: { sink, mode },
  });
  return { onClick: onMouseUp, onMouseDown, onMouseUp };
};

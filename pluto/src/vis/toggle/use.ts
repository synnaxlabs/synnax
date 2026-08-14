// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { useCallback, useEffect } from "react";
import { type z } from "zod";

import { Aether } from "@/aether";
import { useMemoDeepEqual } from "@/memo";
import { toggle } from "@/vis/toggle/aether";

export interface UseProps extends Pick<
  z.input<typeof toggle.toggleStateZ>,
  "source" | "sink" | "stalenessTimeout"
> {
  aetherKey: string;
}

export interface UseReturn extends Pick<
  z.infer<typeof toggle.toggleStateZ>,
  "enabled" | "stale"
> {
  toggle: () => void;
}

export const use = ({
  aetherKey,
  source,
  sink,
  stalenessTimeout,
}: UseProps): UseReturn => {
  const memoProps = useMemoDeepEqual({ source, sink, stalenessTimeout });
  const [, { enabled, stale }, setState, methods] = Aether.use({
    aetherKey,
    type: toggle.Toggle.TYPE,
    schema: toggle.toggleStateZ,
    methods: toggle.toggleMethodsZ,
    initialState: { enabled: false, ...memoProps },
  });
  useEffect(() => {
    setState((state) => ({ ...state, ...memoProps }));
  }, [memoProps, setState]);
  // Wrap to prevent React event from being passed as argument
  const handleToggle = useCallback(() => methods.toggle(), [methods.toggle]);
  return { toggle: handleToggle, enabled, stale };
};

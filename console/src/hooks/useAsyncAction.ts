// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { id } from "@synnaxlabs/x";
import { useMutation } from "@tanstack/react-query";
import { useMemo } from "react";

export interface UseAsyncActionProps {
  key: string;
  action: () => Promise<void>;
}

export const useAsyncAction = ({ key, action }: UseAsyncActionProps) => {
  const addStatus = Status.useAggregator();
  return useMutation({
    mutationKey: [key],
    mutationFn: action,
    onError: (error) => {
      addStatus({
        key: id.id(),
        variant: "error",
        message: error.message,
      });
    },
  }).mutate;
};

export const useAsyncActionMenu = (
  key: string,
  actions: Record<string, () => Promise<void> | void>,
): ((key: string) => void) => {
  const addStatus = Status.useAggregator();
  const mutationKey = useMemo<string[]>(
    () => [key, ...Object.keys(actions)],
    [actions],
  );
  const res = useMutation({
    mutationKey,
    mutationFn: async (key: string) => await actions[key](),
    onError: (error) => {
      addStatus({
        key: id.id(),
        variant: "error",
        message: error.message,
      });
    },
  });
  return (key: string) => res.mutate(key);
};

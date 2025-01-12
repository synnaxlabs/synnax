// Copyright 2024 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Status } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";

export interface UseAsyncActionProps {
  key: string;
  action: () => Promise<void>;
}

export const useAsyncAction = ({ action }: UseAsyncActionProps) => {
  const handleException = Status.useExceptionHandler();
  return useMutation({
    mutationFn: action,
    onError: handleException,
  }).mutate;
};

export const useAsyncActionMenu = (
  actions: Record<string, () => Promise<void> | void>,
): ((key: string) => void) => {
  const handleException = Status.useExceptionHandler();
  const res = useMutation({
    mutationFn: async (key: string) => await actions[key](),
    onError: handleException,
  });
  return (key: string) => res.mutate(key);
};

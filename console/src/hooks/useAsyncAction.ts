import { Status } from "@synnaxlabs/pluto";
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
        key,
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
    onError: (error, key) => {
      addStatus({
        key,
        variant: "error",
        message: error.message,
      });
    },
  });
  return (key: string) => res.mutate(key);
};

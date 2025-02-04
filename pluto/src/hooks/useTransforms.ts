// Copyright 2025 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ArrayTransform, type ArrayTransformPayload } from "@synnaxlabs/x";
import { useCallback, useState } from "react";

export interface ArrayTransformEntry<E> {
  transform: ArrayTransform<E>;
  key: string;
  priority: number;
}

export interface UseTransformsProps<E> {
  transforms?: Array<ArrayTransformEntry<E>>;
}

export interface UseTransformsReturn<E> {
  transform: ArrayTransform<E>;
  setTransform: (key: string, t: ArrayTransform<E>, priority?: number) => void;
  deleteTransform: (key: string) => void;
}

export const useTransforms = <E>({
  transforms: initialTransforms = [],
}: UseTransformsProps<E>): UseTransformsReturn<E> => {
  const [transforms, setTransforms] =
    useState<Array<ArrayTransformEntry<E>>>(initialTransforms);

  const setTransform = useCallback(
    (key: string, t: ArrayTransform<E>, priority = 0): void =>
      setTransforms((prev) => {
        const next = prev.filter((t) => t.key !== key);
        next.push({ key, transform: t, priority });
        next.sort((a, b) => b.priority - a.priority);
        return next;
      }),
    [setTransforms],
  );

  const deleteTransform = useCallback(
    (key: string): void => setTransforms((prev) => prev.filter((t) => t.key !== key)),
    [setTransform],
  );

  const transform = useCallback(
    (props: Omit<ArrayTransformPayload<E>, "transformed">) =>
      transforms.reduce((data, t) => t.transform(data), {
        ...props,
        transformed: false,
      }),
    [transforms],
  );

  return { transform, setTransform, deleteTransform };
};

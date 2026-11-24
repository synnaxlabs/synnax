// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { UnexpectedError, view } from "@synnaxlabs/client";
import { View as PView } from "@synnaxlabs/pluto";
import { useState } from "react";

import {
  useContext as useCtx,
  useFormContext,
  useSearchContext,
} from "@/platform/view/context";

export * from "@/platform/view/filter";
export * from "@/platform/view/Form";
export * from "@/platform/view/Frame";
export * from "@/platform/view/Items";
export * from "@/platform/view/tab";
export * from "@/platform/view/Toolbar";
export * from "@/platform/view/tooltip";

export const useContext = () => useCtx("View.useContext");

type Of<T extends view.Type> = Extract<view.View, { type: T }>;

/**
 * @returns the view that the enclosing frame opens with.
 * @throws {UnexpectedError} if the view is not of the given type.
 */
export const useInitialView = <T extends view.Type>(type: T): Of<T> => {
  const initial = useContext().getInitialView();
  if (initial.type !== type)
    throw new UnexpectedError(`Expected a ${type} view, got a ${initial.type} view`);
  return initial as Of<T>;
};

/**
 * @returns a field of the open view, which must be of the given type, and its setter. A
 * stored view reads the field from the Core and saves a change to it. A static view
 * holds the field while the frame shows.
 */
export const useField = <T extends view.Type, F extends keyof Of<T> & string>(
  type: T,
  field: F,
): [Of<T>[F], (next: Of<T>[F]) => void] => {
  const { selected, staticViews } = useContext();
  const initial = useInitialView(type);
  const stored = !staticViews.includes(selected);
  const [local, setLocal] = useState(initial[field]);
  const live = PView.useResult(stored ? { key: selected } : null).data;
  const { update } = PView.useSetFields();
  if (!stored) return [local, setLocal];
  const value = live?.type === type ? (live as Of<T>)[field] : local;
  return [
    value,
    (next) =>
      update({ key: selected, actions: view.set({ path: [field], value: next }) }),
  ];
};

/** Returns the keys of the selected items of the enclosing form's list. */
export const useSelected = () => useFormContext("View.useSelected").selected;

/** Returns the search term of the enclosing frame, which changes on each key. */
export const useSearch = () => useSearchContext("View.useSearch");

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { context } from "@synnaxlabs/lyra/context";
import { type PropsWithChildren, type ReactElement } from "react";

import {
  type Problem,
  type Rules,
  useForward,
} from "@/platform/notifications/useForward";

const [ProblemsContext, useProblemsContext] = context.create<Problem[]>({
  displayName: "Notifications.ProblemsContext",
  providerName: "Notifications.Provider",
});

/** @returns the open problems of the enclosing {@link Provider}, in no order. */
export const useProblems = (): Problem[] =>
  useProblemsContext("Notifications.useProblems");

/** Props for {@link Provider}. */
export interface ProviderProps extends PropsWithChildren {
  /** The rules of the features that own status keys. */
  rules: Rules;
}

/**
 * Notifies the status writes that the Core streams, by kind. A reply notifies at once
 * unless it is a success, loading, or disabled status. An alarm notifies at once. A
 * state notifies only when it is a warning or an error and its thing is in use. A
 * delayed warning notifies only if its key is still bad 5 s after the problem starts.
 * A problem, which is a key and a since, notifies once. Later writes of the problem add
 * to its count, and the problems open at load never notify. It provides the open
 * problems for {@link useProblems}. Mount it once, in the main window.
 */
export const Provider = ({ rules, children }: ProviderProps): ReactElement => {
  const problems = useForward(rules);
  return <ProblemsContext value={problems}>{children}</ProblemsContext>;
};

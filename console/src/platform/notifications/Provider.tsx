// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type status } from "@synnaxlabs/client";
import { context } from "@synnaxlabs/lyra/context";
import { type PropsWithChildren, type ReactElement, useState } from "react";

/**
 * A warning or error that is open, from a thing in use or a status that no feature
 * owns. A reply is never a problem.
 */
export interface Problem {
  /** The latest bad status of the problem. */
  status: status.Status;
  /** How many bad statuses of the problem the Console got. */
  count: number;
  /** The key of the status whose problem covers this one, such as the task's rack. */
  coveredBy: status.Key | null;
}

const [ProblemsContext, useProblemsContext] = context.create<Problem[]>({
  displayName: "Notifications.ProblemsContext",
  providerName: "Notifications.Provider",
});

const [SetterContext, useSetterContext] = context.create<(next: Problem[]) => void>({
  displayName: "Notifications.SetterContext",
  providerName: "Notifications.Provider",
});

/** @returns the open problems that {@link useForward} tracks, in no order. */
export const useProblems = (): Problem[] =>
  useProblemsContext("Notifications.useProblems");

export const useSetProblems = (): ((next: Problem[]) => void) =>
  useSetterContext("Notifications.useForward");

/** Holds the open problems. Mount it above {@link useForward} and its readers. */
export const Provider = ({ children }: PropsWithChildren): ReactElement => {
  const [problems, setProblems] = useState<Problem[]>([]);
  return (
    <SetterContext value={setProblems}>
      <ProblemsContext value={problems}>{children}</ProblemsContext>
    </SetterContext>
  );
};

// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type CSSProperties, useCallback, useEffect, useState } from "react";

interface Step {
  duration: number;
}

interface Example<S extends Step> {
  steps: S[];
}

interface Timeline<S extends Step> {
  activeTab: number;
  stepIndex: number;
  step: S;
  /** Changes on each resume, so a key that holds it restarts the step animation. */
  playKey: string;
  selectTab: (index: number) => void;
  /** Spread onto the container to pause on hover and to set `--play-state`. */
  containerProps: {
    style: CSSProperties;
    onMouseEnter: () => void;
    onMouseLeave: () => void;
  };
}

/**
 * Plays the steps of the active example in a loop, each for its `duration` in
 * milliseconds. Playback pauses while the pointer is over the container and restarts
 * the current step on leave. Selecting a tab starts its example from the first step.
 */
export const useTimeline = <S extends Step>(examples: Example<S>[]): Timeline<S> => {
  const [activeTab, setActiveTab] = useState(0);
  const [stepIndex, setStepIndex] = useState(0);
  const [paused, setPaused] = useState(false);
  const [tick, setTick] = useState(0);

  const { steps } = examples[activeTab];
  const step = steps[stepIndex];

  useEffect(() => {
    if (paused) return;
    const timeout = setTimeout(
      () => setStepIndex((prev) => (prev + 1) % steps.length),
      step.duration,
    );
    return () => clearTimeout(timeout);
  }, [stepIndex, activeTab, paused, tick, steps.length, step.duration]);

  const selectTab = useCallback(
    (index: number) => {
      if (index === activeTab) return;
      setActiveTab(index);
      setStepIndex(0);
    },
    [activeTab],
  );

  const onMouseEnter = useCallback(() => setPaused(true), []);
  const onMouseLeave = useCallback(() => {
    setPaused(false);
    setTick((t) => t + 1);
  }, []);

  return {
    activeTab,
    stepIndex,
    step,
    playKey: `${activeTab}-${stepIndex}-${tick}`,
    selectTab,
    containerProps: {
      style: { "--play-state": paused ? "paused" : "running" } as CSSProperties,
      onMouseEnter,
      onMouseLeave,
    },
  };
};

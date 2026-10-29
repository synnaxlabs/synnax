// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/github/StarButton.css";

import { useAsyncEffect } from "@synnaxlabs/lyra/hooks";
import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { type AnimationEvent, type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { retrieveStatus, star, URL } from "@/platform/github/star";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

const LEAVE_ANIMATION = "console-github-star-leave";

const Prompt = (): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const starred = Session.GitHub.useSelectStarred();
  const handleError = Status.useErrorHandler();
  // Null until the CLI answers.
  const [cliReady, setCLIReady] = useState<boolean | null>(null);
  const [starring, setStarring] = useState(false);
  // True while a star made from this button plays out, so it outlives the stored star.
  const [celebrating, setCelebrating] = useState(false);
  useAsyncEffect(
    async (signal) => {
      if (starred) return;
      const status = await retrieveStatus();
      if (signal.aborted) return;
      if (status === "starred") dispatch(Session.GitHub.markStarred());
      else setCLIReady(status === "not_starred");
    },
    [starred],
  );
  if ((starred && !celebrating) || cliReady == null) return null;
  const handleClick = async (): Promise<void> => {
    setStarring(true);
    let starredByCLI = false;
    try {
      if (cliReady) starredByCLI = await star();
      if (!starredByCLI) await Runtime.openExternal(URL);
    } catch (e) {
      setStarring(false);
      handleError(e, "Failed to star Synnax on GitHub");
      return;
    }
    setStarring(false);
    setCelebrating(starredByCLI);
    dispatch(Session.GitHub.markStarred());
  };
  const handleAnimationEnd = (e: AnimationEvent): void => {
    if (e.animationName === LEAVE_ANIMATION) setCelebrating(false);
  };
  let label = "Star on GitHub";
  if (starring) label = "Starring";
  if (celebrating) label = "Starred";
  return (
    <button
      type="button"
      className={CSS.cls(
        CSS.B("github-star"),
        starring && CSS.BM("github-star", "starring"),
        celebrating && CSS.BM("github-star", "starred"),
      )}
      disabled={starring || celebrating}
      onClick={() => void handleClick()}
      onAnimationEnd={handleAnimationEnd}
    >
      <span className={CSS.BE("github-star", "field")} aria-hidden />
      <span className={CSS.BE("github-star", "icon")} aria-hidden>
        <Icon.StarOutlined className={CSS.BE("github-star", "outline")} />
        <Icon.StarFilled className={CSS.BE("github-star", "fill")} />
      </span>
      <span className={CSS.BE("github-star", "label")}>{label}</span>
    </button>
  );
};

/**
 * Asks the user to star Synnax on GitHub, in Synnax Desktop only. A click stars it
 * through the GitHub CLI when the CLI is signed in, and opens the repository page
 * otherwise. Renders nothing once the user starred Synnax.
 */
export const StarButton = (): ReactElement | null => (DESKTOP ? <Prompt /> : null);

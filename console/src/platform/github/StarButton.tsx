// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/platform/github/StarButton.css";

import { Icon } from "@synnaxlabs/lyra/icon";
import { Status } from "@synnaxlabs/lyra/status";
import { type AnimationEvent, type ReactElement, useState } from "react";

import { CSS } from "@/platform/css";
import { star, URL } from "@/platform/github/star";
import { Runtime } from "@/platform/runtime";
import { Session } from "@/session";

const LEAVE_ANIMATION = "console-github-star-leave";

/**
 * Asks the user to star Synnax on GitHub. A click stars it through the GitHub CLI when
 * the CLI is signed in, and opens the repository page otherwise. Renders nothing until
 * useSyncStatus checks the CLI, so nothing outside Synnax Desktop, and nothing once the
 * user starred Synnax.
 */
export const StarButton = (): ReactElement | null => {
  const dispatch = Session.useDispatch();
  const starred = Session.GitHub.useSelectStarred();
  const cli = Session.GitHub.useSelectCLI();
  const handleError = Status.useErrorHandler();
  const [starring, setStarring] = useState(false);
  // True while a star made from this button plays out, so it outlives the stored star.
  const [celebrating, setCelebrating] = useState(false);
  if ((starred && !celebrating) || cli === "unchecked") return null;
  const handleClick = async (): Promise<void> => {
    setStarring(true);
    let starredByCLI = false;
    try {
      if (cli === "ready") starredByCLI = await star();
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

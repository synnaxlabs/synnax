// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { type ComponentType, type ReactElement, useEffect, useState } from "react";

interface SceneModule<P> {
  default: ComponentType<P>;
}

/** Keep off-screen geometry out of the HTML and fetch it after island hydration. */
export const DeferredScene = <P extends object>({
  load,
  sceneProps,
  viewBox,
}: {
  load: () => Promise<SceneModule<P>>;
  sceneProps: P;
  viewBox: string;
}): ReactElement => {
  const [module, setModule] = useState<SceneModule<P> | null>(null);
  const [failed, setFailed] = useState(false);
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    let cancelled = false;
    void load().then(
      (loaded) => {
        if (!cancelled) setModule(loaded);
      },
      () => {
        if (!cancelled) setFailed(true);
      },
    );
    return () => {
      cancelled = true;
    };
  }, [load, attempt]);

  if (module != null) return <module.default {...sceneProps} />;
  const [, , width, height] = viewBox.split(" ").map(Number);
  return (
    <div className="scene-load-state" style={{ aspectRatio: `${width} / ${height}` }}>
      {failed ? (
        <button
          type="button"
          onClick={() => {
            setFailed(false);
            setAttempt((previous) => previous + 1);
          }}
        >
          Reload diagram <span aria-hidden="true">↻</span>
        </button>
      ) : (
        <span role="status">Loading diagram</span>
      )}
    </div>
  );
};

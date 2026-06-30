// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/docs/Docs.css";

import { Logo } from "@synnaxlabs/media";
import { Button, Icon, Theming, Triggers } from "@synnaxlabs/pluto";
import { url } from "@synnaxlabs/x";
import { memo, type ReactElement, useEffect, useState } from "react";

import { CSS } from "@/component/css";
import { Session } from "@/session";

const HOST = new url.URL({
  host: "docs.synnaxlabs.com",
  port: 443,
  protocol: "https",
});
export const LAYOUT_TYPE = "docs";

export const LAYOUT: Session.Layout.BaseState = {
  key: LAYOUT_TYPE,
  type: LAYOUT_TYPE,
  location: "mosaic",
  name: "Documentation",
  tab: { editable: false },
};

export const Docs = memo(() => {
  // Iframes prevent drop interactions on the mosaic, so we need to listen for
  // the mouse being held down and add a class the docs that adds a mask over the frame
  // to allow for drop interactions.
  const hover = Triggers.useHeld({
    triggers: [["MouseLeft"]],
    loose: true,
  });

  const [loaded, setLoaded] = useState(false);

  const { theme } = Theming.useContext();

  const { path } = Session.Docs.useSelectLocation();
  const [frameURL, setFrameURL] = useState<url.URL | null>(null);

  const dispatch = Session.useDispatch();

  const handleFrameMessage = (event: Event): void => {
    dispatch(
      Session.Docs.setLocation({
        path: (event as MessageEvent).data.path,
        heading: (event as MessageEvent).data.heading,
      }),
    );
  };

  useEffect(() => {
    const queryParams = {
      noHeader: "true",
      theme: theme.key.includes("dark") ? "dark" : "light",
    };
    setFrameURL(
      HOST.child(path || "reference/console/get-started").child(
        url.buildQueryString(queryParams),
      ),
    );
    window.addEventListener("message", handleFrameMessage);
    return () => window.removeEventListener("message", handleFrameMessage);
  }, []);

  if (frameURL === null) return null;

  return (
    <div className={CSS(CSS.B("docs"), hover.held && CSS.M("hover"))}>
      {!loaded && <Logo.Watermark variant="loader" />}
      <iframe src={frameURL.toString()} onLoad={() => setLoaded(true)} />
    </div>
  );
});
Docs.displayName = "DocsLayoutRenderer";

export const OpenButton = (): ReactElement => {
  const placeLayout = Session.Layout.usePlacer();
  const handleDocs = (): void => {
    placeLayout(LAYOUT);
  };
  return (
    <Button.Button
      size="small"
      variant="text"
      onClick={handleDocs}
      contrast={2}
      className={CSS.BE("docs", "open-button")}
      tooltip="Open Documentation"
    >
      <Icon.QuestionMark />
    </Button.Button>
  );
};

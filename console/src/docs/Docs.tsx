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
import { buildQueryString, URL } from "@synnaxlabs/x";
import { memo, type ReactElement, useEffect, useState } from "react";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useSelectLocation } from "@/docs/selectors";
import { setDocsLocation } from "@/docs/slice";
import { Layout } from "@/layout";

const HOST = new URL({
  host: "docs.synnaxlabs.com",
  port: 443,
  protocol: "https",
});
export const LAYOUT_TYPE = "docs";

export const LAYOUT: Layout.BaseState = {
  key: LAYOUT_TYPE,
  type: LAYOUT_TYPE,
  location: "mosaic",
  name: "Documentation",
  tab: { editable: false },
};

/**
 * Renders a layout that loads the documentation site in an iframe. Updates the docs
 * redux store to preserve the location when re-opening the docs.
 */
export const Docs: Layout.Renderer = memo(() => {
  // Iframes prevent drop interactions on the mosaic, so we need to listen for
  // the mouse being held down and add a class the docs that adds a mask over the frame
  // to allow for drop interactions.
  const hover = Triggers.useHeld({
    triggers: [["MouseLeft"]],
    loose: true,
  });

  const [loaded, setLoaded] = useState(false);

  const { theme } = Theming.useContext();

  const { path } = useSelectLocation();
  const [url, setURL] = useState<URL | null>(null);

  const dispatch = useDispatch();

  const handleFrameMessage = (event: Event): void => {
    dispatch(
      setDocsLocation({
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
    setURL(
      HOST.child(path || "reference/console/get-started").child(
        buildQueryString(queryParams),
      ),
    );
    window.addEventListener("message", handleFrameMessage);
    return () => window.removeEventListener("message", handleFrameMessage);
  }, []);

  if (url === null) return null;

  return (
    <div className={CSS(CSS.B("docs"), hover.held && CSS.M("hover"))}>
      {!loaded && <Logo.Watermark variant="loader" />}
      <iframe src={url.toString()} onLoad={() => setLoaded(true)} />
    </div>
  );
});
Docs.displayName = "DocsLayoutRenderer";

export const OpenButton = (): ReactElement => {
  const placeLayout = Layout.usePlacer();
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

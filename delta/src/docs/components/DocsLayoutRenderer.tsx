// Copyright 2023 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { memo, useEffect, useState } from "react";

import { Theming, Triggers } from "@synnaxlabs/pluto";
import { URL, buildQueryString } from "@synnaxlabs/x";
import { useDispatch } from "react-redux";

import { CSS } from "@/css";
import { useSelectDocsLocation } from "@/docs/hooks";
import { setDocsLocation } from "@/docs/store";
import { LayoutRenderer } from "@/layout";

import "./DocsLayoutRenderer.css";

const DOCS_HOST = new URL({
  host: "docs.synnaxlabs.com",
  port: 443,
  protocol: "https",
});

/**
 * Renders a layout that loads the documentation site in an iframe. Updates the docs
 * redux store to preserve the location when re-opening the docs.
 */
export const DocsLayoutRenderer: LayoutRenderer = memo(() => {
  // Iframes prevent drop interactions on the mosaic, so we need to listen for
  // the mouse being held down and add a class the docs that adds a mask over the frame
  // to allow for drop interactions.
  const hover = Triggers.useHeld([["MouseLeft"]]);

  const { theme } = Theming.useContext();

  const { path } = useSelectDocsLocation();
  const [url, setURL] = useState<URL | null>(null);

  const dispatch = useDispatch();

  const handleFrameMessage = (event: Event): void => {
    dispatch(
      setDocsLocation({
        path: (event as MessageEvent).data.path,
        heading: (event as MessageEvent).data.heading,
      })
    );
  };

  useEffect(() => {
    const queryParams = {
      noHeader: true,
      theme: theme.key.includes("dark") ? "dark" : "light",
    };
    setURL(DOCS_HOST.child(path).child(buildQueryString(queryParams)));
    window.addEventListener("message", handleFrameMessage);
    return () => window.removeEventListener("message", handleFrameMessage);
  }, []);

  if (url === null) return null;

  return (
    <div className={CSS(CSS.B("docs"), hover.held && CSS.M("hover"))}>
      <iframe src={url.toString()} />
    </div>
  );
});
DocsLayoutRenderer.displayName = "DocsLayoutRenderer";

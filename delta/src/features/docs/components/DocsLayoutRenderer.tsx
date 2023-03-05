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

import { setDocsLocation } from "../store";
import { useSelectDocsLocation } from "../store/selectors";

import { CSS } from "@/css";
import { LayoutRenderer } from "@/features/layout";

import "./DocsLayoutRenderer.css";

const DOCS_HOST = new URL({ host: "localhost", port: 3000, protocol: "http" });

export const DocsLayoutRenderer: LayoutRenderer = memo(() => {
  const hover = Triggers.useHeld([["MouseLeft", null]]);

  const { theme } = Theming.useContext();

  const { path } = useSelectDocsLocation();
  const [url, setUrl] = useState<URL | null>(null);

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
    const url = DOCS_HOST.child(path).child(buildQueryString(queryParams));
    setUrl(url);
    window.addEventListener("message", handleFrameMessage);
    return () => {
      window.removeEventListener("message", handleFrameMessage);
    };
  }, []);

  if (url === null) return null;

  return (
    <div className={CSS(CSS.B("docs"), hover.held && CSS.M("hover"))}>
      <iframe src={url.toString()} />
    </div>
  );
});
DocsLayoutRenderer.displayName = "DocsLayoutRenderer";

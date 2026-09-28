// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type FC, type ReactElement, useCallback, useState } from "react";

interface SDKShowcaseProps {
  codeHtmls: string[][];
}

const LANG_TABS: { title: string; icon: FC }[] = [
  { title: "Python", icon: Icon.Python },
  { title: "TypeScript", icon: Icon.TypeScript },
  { title: "C++", icon: Icon.CPlusPlus },
];

const OP_LABELS = ["Stream", "Write", "Read"];

export const SDKShowcase = ({ codeHtmls }: SDKShowcaseProps): ReactElement => {
  const [activeLang, setActiveLang] = useState(0);

  const handleLangClick = useCallback(
    (i: number) => {
      if (i !== activeLang) setActiveLang(i);
    },
    [activeLang],
  );

  return (
    <div className="sdks-showcase">
      <div className="viz-tabs">
        {LANG_TABS.map(({ title, icon: TabIcon }, i) => (
          <button
            key={title}
            className={`viz-tab${i === activeLang ? " viz-tab--active" : ""}`}
            onClick={() => handleLangClick(i)}
          >
            <TabIcon />

            {title}
          </button>
        ))}
      </div>
      <div className="sdks-panels">
        {OP_LABELS.map((label, opIdx) => (
          <div key={label} className="sdks-panel">
            <span className="sdks-panel-label">{label}</span>
            <div
              className="code-panel"
              dangerouslySetInnerHTML={{
                __html: codeHtmls[activeLang][opIdx],
              }}
            />
          </div>
        ))}
      </div>
    </div>
  );
};

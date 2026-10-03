// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import { Icon } from "@synnaxlabs/lyra/icon";
import { type ReactElement, useCallback, useState } from "react";

import { type VizTab, VizTabs } from "@/components/common/VizTabs";

interface SDKShowcaseProps {
  codeHtmls: string[][];
}

const LANG_TABS: VizTab[] = [
  { key: "python", title: "Python", icon: Icon.Python },
  { key: "typescript", title: "TypeScript", icon: Icon.TypeScript },
  { key: "cpp", title: "C++", icon: Icon.CPlusPlus },
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
      <VizTabs tabs={LANG_TABS} active={activeLang} onSelect={handleLangClick} />
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

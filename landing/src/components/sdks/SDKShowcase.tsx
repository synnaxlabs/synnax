import { Icon } from "@synnaxlabs/pluto";
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
      <div className="automate-viz-tabs">
        {LANG_TABS.map(({ title, icon: TabIcon }, i) => (
          <button
            key={title}
            className={`automate-viz-tab${i === activeLang ? " automate-viz-tab--active" : ""}`}
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
              className="automate-code-panel"
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

import { type ReactElement, useMemo } from "react";

interface CodePanelProps {
  html: string;
  activeLines: number[];
  className?: string;
}

const LINE_TAG = '<span class="line">';
const ACTIVE_LINE_TAG = '<span class="line" data-active="true">';

export const CodePanel = ({
  html,
  activeLines,
  className,
}: CodePanelProps): ReactElement => {
  const processedHtml = useMemo(() => {
    let lineIndex = 0;
    return html.replace(/<span class="line">/g, () => {
      lineIndex++;
      return activeLines.includes(lineIndex) ? ACTIVE_LINE_TAG : LINE_TAG;
    });
  }, [html, activeLines]);

  return (
    <div
      className={`code-panel ${className ?? ""}`}
      dangerouslySetInnerHTML={{ __html: processedHtml }}
    />
  );
};

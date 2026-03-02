import { type ReactElement, useEffect, useRef } from "react";

interface CodePanelProps {
  html: string;
  activeLines: number[];
  className?: string;
}

export const CodePanel = ({
  html,
  activeLines,
  className,
}: CodePanelProps): ReactElement => {
  const ref = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (ref.current == null) return;
    const lines = ref.current.querySelectorAll(".line");
    lines.forEach((line, i) => {
      const lineNum = i + 1;
      if (activeLines.includes(lineNum)) line.setAttribute("data-active", "true");
      else line.removeAttribute("data-active");
    });
  }, [activeLines, html]);

  return (
    <div
      ref={ref}
      className={`code-panel ${className ?? ""}`}
      dangerouslySetInnerHTML={{ __html: html }}
    />
  );
};

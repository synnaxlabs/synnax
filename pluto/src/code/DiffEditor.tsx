// Copyright 2026 Synnax Labs, Inc.
//
// Use of this software is governed by the Business Source License included in the file
// licenses/BSL.txt.
//
// As of the Change Date specified in that file, in accordance with the Business Source
// License, use of this software will be governed by the Apache License, Version 2.0,
// included in the file licenses/APL.txt.

import "@/code/DiffEditor.css";

import { Flex } from "@synnaxlabs/lyra/flex";
import { Status } from "@synnaxlabs/lyra/status";
import { Text } from "@synnaxlabs/lyra/text";
import { type ReactElement, type ReactNode, useEffect, useRef } from "react";

import {
  disableMonacoCommandPalette,
  forwardGlobalTriggers,
  useTheme,
} from "@/code/embed";
import { type Monaco, useLanguage, useMonaco } from "@/code/Provider";
import { CSS } from "@/css";
import { Errors } from "@/errors";

// The split stays at half so the labels above each side line up with it.
const OPTIONS: Monaco.editor.IStandaloneDiffEditorConstructionOptions = {
  automaticLayout: true,
  readOnly: true,
  domReadOnly: true,
  originalEditable: false,
  enableSplitViewResizing: false,
  useInlineViewWhenSpaceIsLimited: false,
  renderOverviewRuler: false,
  renderMarginRevertIcon: false,
  renderGutterMenu: false,
  hideUnchangedRegions: { enabled: true, contextLineCount: 3 },
  minimap: { enabled: false },
  lineNumbersMinChars: 3,
  folding: false,
  links: false,
  contextmenu: false,
  scrollBeyondLastLine: false,
  wordWrap: "off",
  renderLineHighlight: "none",
};

export interface DiffEditorProps extends Omit<Flex.BoxProps, "children"> {
  /** original is the text before the change. */
  original: string;
  /** modified is the text after the change. */
  modified: string;
  /** language is the id of a language registered with Code.Provider, or a Monaco
   * built-in such as "json". */
  language: string;
  /** originalLabel names the original text above its side. */
  originalLabel?: ReactNode;
  /** modifiedLabel names the modified text above its side. */
  modifiedLabel?: ReactNode;
  /** inline shows both texts in one column instead of side by side. */
  inline?: boolean;
  loading?: ReactNode;
}

interface LabelProps {
  children: ReactNode;
}

const Label = ({ children }: LabelProps): ReactElement => (
  <Text.Text
    level="small"
    color={9}
    overflow="fade"
    className={CSS.BE("diff-editor", "label")}
  >
    {children}
  </Text.Text>
);

const DiffEditorInternal = ({
  original,
  modified,
  language,
  originalLabel,
  modifiedLabel,
  inline = false,
  className,
  background = 1,
  ...rest
}: Omit<DiffEditorProps, "loading">): ReactElement => {
  const containerRef = useRef<HTMLDivElement>(null);
  const monaco = useMonaco();
  const languageDef = useLanguage(language);
  const theme = useTheme(languageDef?.theme != null);

  useEffect(() => {
    const container = containerRef.current;
    if (container == null) return;
    const editor = monaco.editor.createDiffEditor(container, {
      ...OPTIONS,
      theme,
      renderSideBySide: !inline,
    });
    const originalModel = monaco.editor.createModel(original, language);
    const modifiedModel = monaco.editor.createModel(modified, language);
    editor.setModel({ original: originalModel, modified: modifiedModel });
    disableMonacoCommandPalette(monaco);
    const forwards = [
      forwardGlobalTriggers(editor.getOriginalEditor()),
      forwardGlobalTriggers(editor.getModifiedEditor()),
    ];
    return () => {
      forwards.forEach((d) => d.dispose());
      editor.dispose();
      originalModel.dispose();
      modifiedModel.dispose();
    };
  }, [monaco, original, modified, language, inline]);

  useEffect(() => {
    monaco.editor.setTheme(theme);
  }, [monaco, theme]);

  const hasLabels = originalLabel != null || modifiedLabel != null;
  return (
    <Flex.Box
      y
      grow
      empty
      background={background}
      {...rest}
      className={CSS.cls(className, CSS.B("diff-editor"))}
    >
      {hasLabels && (
        <Flex.Box x empty className={CSS.BE("diff-editor", "labels")}>
          {originalLabel != null && <Label>{originalLabel}</Label>}
          {modifiedLabel != null && <Label>{modifiedLabel}</Label>}
        </Flex.Box>
      )}
      <Flex.Box ref={containerRef} grow className={CSS.BE("diff-editor", "body")} />
    </Flex.Box>
  );
};

const LOADING = (
  <Status.Loading>
    <Status.Orbital />
  </Status.Loading>
);

/**
 * DiffEditor shows two texts side by side in read-only Monaco editors, marking what
 * changed between them and folding the unchanged regions. It suspends while the Monaco
 * runtime initializes on first use, rendering `loading` until ready.
 */
export const DiffEditor = ({
  loading = LOADING,
  ...props
}: DiffEditorProps): ReactElement => (
  <Errors.SuspenseBoundary loading={loading}>
    <DiffEditorInternal {...props} />
  </Errors.SuspenseBoundary>
);

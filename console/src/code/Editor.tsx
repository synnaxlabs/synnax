import { Icon } from "@synnaxlabs/media";
import { Align, Button, Input, Theming } from "@synnaxlabs/pluto";
import * as monaco from "monaco-editor";
import { useEffect, useRef } from "react";

import { Layout } from "@/layout";

export const EDITOR_LAYOUT_TYPE = "code_editor";

export const createEditorLayout = ({
  name,
  window,
  ...rest
}: Partial<Layout.State>): Layout.State => ({
  ...rest,
  key: EDITOR_LAYOUT_TYPE,
  type: EDITOR_LAYOUT_TYPE,
  windowKey: EDITOR_LAYOUT_TYPE,
  icon: "Range",
  location: "mosaic",
  name: name ?? "Range.Create",
});

export const EDITOR_SELECTABLE: Layout.Selectable = {
  key: EDITOR_LAYOUT_TYPE,
  title: "Code Editor",
  icon: <Icon.Add />,
  create: (layoutKey) => createEditorLayout({ key: layoutKey }),
};

export interface EditorProps
  extends Input.Control<string>,
    Omit<Align.SpaceProps, "value" | "onChange"> {}

export const Editor = ({
  value,
  onChange,
  className,
  style,
  ...props
}: EditorProps) => {
  const editorRef = useRef<HTMLDivElement | null>(null); // A ref to store the editor DOM element
  const monacoRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null); // A ref to store the Monaco editor instance
  const theme = Theming.use();

  useEffect(() => {
    if (editorRef.current === null) return;
    monaco.editor.defineTheme("vs-dark-custom", {
      base: "vs-dark",
      inherit: true,
      rules: [],
      colors: {
        "editor.background": theme.colors.gray.l1.hex,
        "editor.foreground": theme.colors.gray.l9.hex,
        "editor.selectionBackground": theme.colors.gray.l1.hex,
        "editor.lineHighlightBackground": theme.colors.gray.l2.hex,
        "editorCursor.foreground": theme.colors.primary.z.hex,
        "editorWhitespace.foreground": theme.colors.gray.l2.hex,
      },
    });
    monacoRef.current = monaco.editor.create(editorRef.current, {
      value,
      language: "python",
      theme: "vs-dark-custom",
      automaticLayout: true,
    });
    monacoRef.current.onDidChangeModelContent(() => {
      if (monacoRef.current === null) return;
      onChange(monacoRef.current.getValue());
    });
    return () => {
      if (monacoRef.current) monacoRef.current.dispose();
    };
  }, []);

  return (
    <Align.Space direction="y" grow style={{ height: "100%", ...style }} {...props}>
      <div ref={editorRef} style={{ height: "100%", position: "relative" }} />
    </Align.Space>
  );
};

export const EditorLayout: Layout.Renderer = ({ layoutKey }) => {};

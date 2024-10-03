import { Icon } from "@synnaxlabs/media";
import { Align, Button } from "@synnaxlabs/pluto";
import { useMutation } from "@tanstack/react-query";
import * as monaco from "monaco-editor";
import React, { useEffect, useRef } from "react";
import { useDispatch } from "react-redux";

import { Layout } from "@/layout";
import { setXChannel, setYChannels } from "@/lineplot/slice";

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

export const Editor: Layout.Renderer = ({layoutKey}) => {
  const editorRef = useRef<HTMLDivElement | null>(null); // A ref to store the editor DOM element
  const monacoRef = useRef<monaco.editor.IStandaloneCodeEditor | null>(null); // A ref to store the Monaco editor instance
  const layouts = Layout.useSelectMany();
  const plot = layouts.find((layout) => layout.type === "lineplot");
  const d = useDispatch();
  const layout = Layout.useSelect(layoutKey);

  
  useEffect(() => {
    if (editorRef.current === null) return;
    monacoRef.current = monaco.editor.create(editorRef.current, {
      value: `# Write your Python code here\nprint("Hello, World!")`,
      language: "python",
      theme: "vs-dark",
      automaticLayout: true,
    });
    return () => {
      if (monacoRef.current) monacoRef.current.dispose();
    };
  }, []);

  const handlePlay = useMutation({
    mutationKey: ["play", plot?.key, layout?.name],
    mutationFn: async () => {
      if (monacoRef.current === null || layout == null) return;
      const code = monacoRef.current.getValue();
      try {
        await fetch("http://127.0.0.1:5000/api/v1/create_calculated", {
          method: "POST",
          body: JSON.stringify({ 
            name: layout.name,
            statement: code,
          }),
          headers: {
            "Content-Type": "application/json",
          },
        });
      } catch (error) {
        console.error(error);
      }
    },
  });

  return (
    <Align.Space direction="y" grow style={{ height: "100%" }}>
      <div ref={editorRef} style={{ height: "100%", position: "relative" }} />
      <Button.Icon
        onClick={handlePlay.mutate}
        style={{ position: "absolute", zIndex: 500 }}
      >
        <Icon.Play />
      </Button.Icon>
    </Align.Space>
  );
};

import * as monaco from 'monaco-editor';
import React, { useEffect, useRef } from 'react';

export const Editor = () => {
  const editorRef = useRef(null); // A ref to store the editor DOM element
  const monacoRef = useRef(null); // A ref to store the Monaco editor instance

  useEffect(() => {
    monacoRef.current = monaco.editor.create(editorRef.current, {
      value: `# Write your Python code here\nprint("Hello, World!")`,
      language: 'python',
      theme: 'vs-dark',
      automaticLayout: true,
    });
    return () => {
      if (monacoRef.current) 
        monacoRef.current.dispose();
    };
  }, []);

  return (
    <div style={{ height: '100vh', display: 'flex', flexDirection: 'column' }}>
      <div
        ref={editorRef}
        style={{ height: '90vh', border: '1px solid grey' }}
      ></div>
      <button
        style={{ padding: '10px', fontSize: '16px', marginTop: '10px' }}
        onClick={() => {
          const editorValue = monacoRef.current.getValue();
          console.log('Python Code:', editorValue);
          // Here you can pass the Python code to a backend or run it with Pyodide
        }}
      >
        Log Python Code
      </button>
    </div>
  );
};

export default App;

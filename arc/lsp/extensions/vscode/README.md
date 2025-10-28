# Arc Language Support for VSCode

This extension provides language support for the Arc programming language, including
syntax highlighting, code completion, hover information, and error diagnostics.

## Features

- **Syntax Highlighting**: Full syntax highlighting for Arc language constructs
- **Code Completion**: Context-aware code completion for keywords, types, and built-in
  functions
- **Hover Information**: Documentation on hover for language elements
- **Error Diagnostics**: Real-time syntax error reporting
- **Auto-formatting**: Automatic code formatting (coming soon)
- **Go to Definition**: Navigate to function and variable definitions (coming soon)

## Requirements

The Arc Language Server (`arc-lsp`) must be installed and available in your PATH, or you
can configure the path in the extension settings.

### Building the Language Server

```bash
cd arc/lsp
go build -o arc-lsp
```

## Extension Settings

This extension contributes the following settings:

- `arc.lsp.path`: Path to the Arc LSP executable (default: `arc-lsp`)
- `arc.lsp.debug`: Enable debug logging for the Arc LSP (default: `false`)
- `arc.lsp.logFile`: Path to the LSP log file, empty for stderr (default: `""`)

## Installation

### From Source

1. Build the LSP server:

   ```bash
   cd arc/lsp
   go build -o arc-lsp
   ```

2. Install extension dependencies:

   ```bash
   cd arc/lsp/extensions/vscode
   npm install
   ```

3. Compile the extension:

   ```bash
   npm run compile
   ```

4. Package the extension:

   ```bash
   npx vsce package
   ```

5. Install the generated `.vsix` file in VSCode:
   - Open VSCode
   - Go to Extensions view (Ctrl+Shift+X)
   - Click on the "..." menu
   - Select "Install from VSIX..."
   - Choose the generated `.vsix` file

## Development

To develop the extension:

1. Open the `arc/lsp/extensions/vscode` folder in VSCode
2. Run `npm install` to install dependencies
3. Press F5 to launch a new VSCode window with the extension loaded
4. Open a `.arc` file to test the extension

## Known Issues

- Go to Definition is not yet implemented
- Document symbols are not yet implemented
- Auto-formatting is not yet implemented

## Release Notes

### 0.1.0

Initial release with basic language support:

- Syntax highlighting
- Code completion
- Hover information
- Error diagnostics

# VS Code configuration

VS Code is tricky to set up with a monorepo like Synnax. The editor can have trouble
detecting package installations like ESLint and TypeScript, run into issues with Python
virtual environments, etc. Luckily, VS Code has a feature called workspaces that lets us
open different directories under separate contexts/configurations.

Our workspace file is located at
[`.vscode/synnax.code-workspace`](../../.vscode/synnax.code-workspace). To open Synnax
properly in code, click the `File` menu, then the `Open Workspace From File` option.
Navigate to the `.vscode` directory in the root of the repository and select the
`synnax.code-workspace` file.

When you open the workspace for the first time, you'll see a notification that prompts
you to install the recommended extensions. We suggest you install all of these.

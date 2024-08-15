import { Status } from "@synnaxlabs/pluto";

export const useCopyToClipboard = (): ((text: string, name: string) => void) => {
  const addStatus = Status.useAggregator();
  return (text: string, name: string) => {
    navigator.clipboard
      .writeText(text)
      .then(() => {
        addStatus({
          variant: "success",
          message: `Copied ${name} to clipboard.`,
        });
      })
      .catch((e) => {
        addStatus({
          variant: "error",
          message: `Failed to copy ${name} to clipboard.`,
          description: e.message,
        });
      });
  };
};

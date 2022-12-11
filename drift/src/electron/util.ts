export const isRenderer = () => {
  if (!process) return true;
  const tProcess = process as unknown as { type: string };
  return !tProcess.type || tProcess.type === "renderer";
};

export const actionEvent = "drift://action";
export const driftKeyArgv = "driftKey:";

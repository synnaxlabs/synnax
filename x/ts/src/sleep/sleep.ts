import { CrudeTimeSpan, TimeSpan } from "@/telem";

export const sleep = async (span: CrudeTimeSpan): Promise<void> =>
  await new Promise((resolve) =>
    setTimeout(resolve, TimeSpan.fromMilliseconds(span).milliseconds),
  );

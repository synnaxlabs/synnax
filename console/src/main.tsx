import { createRoot } from "react-dom/client";

import { Main } from "@/Root";

const rootEl = document.getElementById("root") as HTMLElement;

createRoot(rootEl).render(<Main />);

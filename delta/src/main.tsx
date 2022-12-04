import { StrictMode } from "react";
import ReactDOM from "react-dom/client";
import { App } from "./App";
import { Provider as DriftProvider } from "@synnaxlabs/drift";
import { store } from "./store";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
	<StrictMode>
		<DriftProvider store={store}>
			<App />
		</DriftProvider>
	</StrictMode>
);

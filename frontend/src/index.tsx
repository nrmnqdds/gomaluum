import React from "react";
import ReactDOM from "react-dom/client";
import App from "./App";
import "./index.css";
import { Helmet, HelmetProvider } from "react-helmet-async";

const app = (
	<HelmetProvider>
		<React.StrictMode>
			<App />
			<Helmet>
				<title>QudStack</title>
			</Helmet>
		</React.StrictMode>
	</HelmetProvider>
);

const rootEl = document.querySelector("#root");

if (rootEl) {
	ReactDOM.hydrateRoot(rootEl, app);
}

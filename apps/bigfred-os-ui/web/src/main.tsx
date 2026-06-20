import React from "react";
import ReactDOM from "react-dom/client";

import "@fontsource/roboto/400.css";
import "@fontsource/roboto/500.css";
import "@fontsource/roboto/700.css";
import "@fontsource/jetbrains-mono/400.css";

import App from "./App";
import "./index.css";

const root = document.getElementById("root");
if (!root) {
  throw new Error("#root not found");
}

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);

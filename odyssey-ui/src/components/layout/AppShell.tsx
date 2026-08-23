import { useState } from "react";

import { Sidebar } from "./Sidebar";
import { Dashboard } from "../../pages/Dashboard";
import Executions from "../../pages/ExecutionPage";

export function AppShell() {
  const [activePage, setActivePage] = useState("Dashboard");

  const renderPage = () => {
    switch (activePage) {
      case "Executions":
        return <Executions />;

      case "Dashboard":
      default:
        return <Dashboard />;
    }
  };

  return (
    <div className="app-shell">
      <Sidebar
        activePage={activePage}
        onNavigate={setActivePage}
      />

      <main className="main-content">
        {renderPage()}
      </main>
    </div>
  );
}
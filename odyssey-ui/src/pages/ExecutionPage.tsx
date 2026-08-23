import { useMemo, useState } from "react";

import ExecutionFilters from "../components/executions/ExecutionFilters";
import ExecutionList from "../components/executions/ExecutionList";
import ExecutionSummary from "../components/executions/ExecutionSummary";
import type { Execution } from "../components/executions/ExecutionRow";

const mockExecutions: Execution[] = [
  {
    id: "exec_7f3a2e",
    key: "customer_123",
    target: "payment",
    function: "process_payment",
    status: "completed",
    attempt: 1,
    startedAt: "2m ago",
    duration: "421ms",
  },
  {
    id: "exec_9b42c1",
    key: "customer_123",
    target: "payment",
    function: "charge_payment",
    status: "executing",
    attempt: 2,
    startedAt: "8s ago",
    duration: "12s",
  },
  {
    id: "exec_3a91d8",
    key: "customer_123",
    target: "notification",
    function: "send_receipt",
    status: "claimed",
    attempt: 1,
    startedAt: "11s ago",
  },
  {
    id: "exec_1c72f4",
    key: "order_789",
    target: "fulfillment",
    function: "update_ledger",
    status: "failed",
    attempt: 3,
    startedAt: "3m ago",
    duration: "1.2s",
  },
  {
    id: "exec_5e83a1",
    key: "order_456",
    target: "payment",
    function: "refund_payment",
    status: "uncertain",
    attempt: 1,
    startedAt: "5m ago",
    duration: "2.4s",
  },
  {
    id: "exec_8d14b6",
    key: "customer_892",
    target: "order",
    function: "create_order",
    status: "completed",
    attempt: 1,
    startedAt: "7m ago",
    duration: "183ms",
  },
  {
    id: "exec_4f29c7",
    key: "customer_441",
    target: "notification",
    function: "send_confirmation",
    status: "reconciling",
    attempt: 2,
    startedAt: "9m ago",
    duration: "4.8s",
  },
  {
    id: "exec_6b51e9",
    key: "order_331",
    target: "order",
    function: "update_order",
    status: "queued",
    attempt: 1,
    startedAt: "12m ago",
  },
];

export default function Executions() {
  const [search, setSearch] = useState("");
  const [status, setStatus] = useState("all");
  const [target, setTarget] = useState("all");
  const [keyName, setKeyName] = useState("all");

  const filteredExecutions = useMemo(() => {
    return mockExecutions.filter((execution) => {
      const searchValue = search.toLowerCase();

      const matchesSearch =
        !searchValue ||
        execution.id.toLowerCase().includes(searchValue) ||
        execution.key.toLowerCase().includes(searchValue) ||
        execution.target.toLowerCase().includes(searchValue) ||
        execution.function.toLowerCase().includes(searchValue);

      const matchesStatus =
        status === "all" || execution.status === status;

      const matchesTarget =
        target === "all" || execution.target === target;

      const matchesKey =
        keyName === "all" || execution.key === keyName;

      return (
        matchesSearch &&
        matchesStatus &&
        matchesTarget &&
        matchesKey
      );
    });
  }, [search, status, target, keyName]);

  const summary = useMemo(() => {
    return {
      total: mockExecutions.length,

      running: mockExecutions.filter(
        (execution) =>
          execution.status === "executing" ||
          execution.status === "claimed"
      ).length,

      completed: mockExecutions.filter(
        (execution) => execution.status === "completed"
      ).length,

      failed: mockExecutions.filter(
        (execution) => execution.status === "failed"
      ).length,

      uncertain: mockExecutions.filter(
        (execution) =>
          execution.status === "uncertain" ||
          execution.status === "reconciling"
      ).length,
    };
  }, []);

  const handleExecutionClick = (execution: Execution) => {
    console.log("Open execution:", execution.id);
  };

  return (
    <div className="executions">
      <header className="page-header">
        <div>
          <h1>Executions</h1>

          <p>
            Monitor every execution across your keys and targets.
          </p>
        </div>

        <div className="agent-status">
          <span className="status-dot" />
          Agent
          <span className="connection-badge">
            Connected
          </span>
        </div>
      </header>

      <ExecutionFilters
        search={search}
        status={status}
        target={target}
        keyName={keyName}
        onSearchChange={setSearch}
        onStatusChange={setStatus}
        onTargetChange={setTarget}
        onKeyChange={setKeyName}
      />

      <ExecutionSummary
        total={summary.total}
        running={summary.running}
        completed={summary.completed}
        failed={summary.failed}
        uncertain={summary.uncertain}
      />

      <ExecutionList
        executions={filteredExecutions}
        onExecutionClick={handleExecutionClick}
      />
    </div>
  );
}
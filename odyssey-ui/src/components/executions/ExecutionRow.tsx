type ExecutionStatus =
  | "queued"
  | "claimed"
  | "executing"
  | "completed"
  | "failed"
  | "uncertain"
  | "reconciling";

export type Execution = {
  id: string;
  key: string;
  target: string;
  function: string;
  status: ExecutionStatus;
  attempt: number;
  startedAt: string;
  duration?: string;
};

type ExecutionRowProps = {
  execution: Execution;
  onClick?: (execution: Execution) => void;
};

function getStatusClass(status: ExecutionStatus) {
  switch (status) {
    case "completed":
      return "status-success";

    case "failed":
      return "status-error";

    case "uncertain":
    case "reconciling":
      return "status-warning";

    default:
      return "status-neutral";
  }
}

function formatStatus(status: ExecutionStatus) {
  return status.toUpperCase();
}

export default function ExecutionRow({
  execution,
  onClick,
}: ExecutionRowProps) {
  return (
    <button
      type="button"
      className="execution-row"
      onClick={() => onClick?.(execution)}
    >
      <div className="execution-main">
        <div className="execution-function">{execution.function}</div>

        <div className="execution-meta">
          {execution.key} / {execution.target}
        </div>
      </div>

      <div className={`status ${getStatusClass(execution.status)}`}>
        <span className="status-dot-small" />
        {formatStatus(execution.status)}
      </div>

      <div className="execution-attempt">
        {execution.attempt}
      </div>

      <div className="execution-started">
        {execution.startedAt}
      </div>

      <div className="execution-duration">
        {execution.duration ?? "—"}
      </div>
    </button>
  );
}
type ExecutionSummaryProps = {
  total: number;
  running: number;
  completed: number;
  failed: number;
  uncertain: number;
};

type SummaryItemProps = {
  label: string;
  value: number;
  className?: string;
};

function SummaryItem({
  label,
  value,
  className = "",
}: SummaryItemProps) {
  return (
    <div className="execution-summary-item">
      <div className="execution-summary-label">
        {label}
      </div>

      <div className={`execution-summary-value ${className}`}>
        {value}
      </div>
    </div>
  );
}

export default function ExecutionSummary({
  total,
  running,
  completed,
  failed,
  uncertain,
}: ExecutionSummaryProps) {
  return (
    <section className="execution-summary odyssey-panel">
      <SummaryItem
        label="ALL"
        value={total}
      />

      <SummaryItem
        label="RUNNING"
        value={running}
        className="success"
      />

      <SummaryItem
        label="COMPLETED"
        value={completed}
        className="success"
      />

      <SummaryItem
        label="FAILED"
        value={failed}
        className="error"
      />

      <SummaryItem
        label="UNCERTAIN"
        value={uncertain}
        className="warning"
      />
    </section>
  );
}
export function Dashboard() {
  return (
    <div className="dashboard">
      <header className="page-header">
        <div>
          <h1>Dashboard</h1>
          <p>System overview</p>
        </div>

        <div className="agent-status">
          <span className="status-dot" />
          <span>Agent</span>
          <span className="connection-badge">CONNECTED</span>
        </div>
      </header>

      {/* Summary */}
      <section className="summary-panel odyssey-panel">
        <SummaryItem
          label="TOTAL KEYS"
          value="18"
          detail="↑ 2 today"
          tone="success"
        />

        <SummaryItem
          label="TOTAL TARGETS"
          value="64"
          detail="↑ 7 today"
          tone="success"
        />

        <SummaryItem
          label="EXECUTING"
          value="5"
          detail="● running"
          tone="success"
        />

        <SummaryItem
          label="WAITING"
          value="12"
          detail="● waiting"
          tone="warning"
        />

        <SummaryItem
          label="COMPLETED"
          value="47"
          detail="↑ 9 today"
          tone="success"
        />
      </section>

      {/* Main panels */}
      <section className="dashboard-grid">
        <KeysPanel />
        <ActivityPanel />
      </section>

      {/* Recent targets */}
      <RecentTargets />
    </div>
  );
}

function SummaryItem({
  label,
  value,
  detail,
  tone,
}: {
  label: string;
  value: string;
  detail: string;
  tone: "success" | "warning";
}) {
  return (
    <div className="summary-item">
      <div className="summary-label">{label}</div>
      <div className="summary-value">{value}</div>
      <div className={`summary-detail ${tone}`}>
        {detail}
      </div>
    </div>
  );
}

function KeysPanel() {
  const keys = [
    { key: "user:123", targets: "3 targets", status: "EXECUTING", tone: "success" },
    { key: "user:456", targets: "2 targets", status: "WAITING", tone: "warning" },
    { key: "order:789", targets: "4 targets", status: "EXECUTING", tone: "success" },
    { key: "invoice:001", targets: "1 target", status: "COMPLETED", tone: "neutral" },
    { key: "email:campaign", targets: "5 targets", status: "WAITING", tone: "warning" },
  ];

  return (
    <section className="panel odyssey-panel">
      <PanelHeader title="KEYS" action="View all →" />

      <div className="key-list">
        {keys.map((item) => (
          <div className="key-row" key={item.key}>
            <div className="key-icon">⚿</div>

            <div className="key-info">
              <div className="key-name">{item.key}</div>
              <div className="key-targets">{item.targets}</div>
            </div>

            <Status
              label={item.status}
              tone={item.tone}
            />
          </div>
        ))}
      </div>

      <button className="text-button">+ New Key</button>
    </section>
  );
}

function ActivityPanel() {
  const activity = [
    ["13:42:10", "user:123 / charge-card", "execution started", "success"],
    ["13:42:11", "user:123 / send-email", "input validated", "success"],
    ["13:42:12", "user:456 / generate-pdf", "waiting for lock", "warning"],
    ["13:42:13", "order:789 / charge-card", "error: rate limited", "error"],
    ["13:42:15", "order:789 / charge-card", "retrying (attempt 2)", "warning"],
    ["13:42:18", "invoice:001 / send-email", "completed in 1.2s", "success"],
    ["13:42:20", "user:123 / audit-log", "execution started", "success"],
  ];

  return (
    <section className="panel odyssey-panel">
      <PanelHeader title="ACTIVITY (LIVE)" />

      <div className="activity-list">
        {activity.map(([time, target, message, tone]) => (
          <div className="activity-row" key={`${time}-${target}`}>
            <span className="activity-time">{time}</span>

            <span className={`activity-dot ${tone}`} />

            <span className="activity-target">{target}</span>

            <span className="activity-message">{message}</span>
          </div>
        ))}
      </div>

      <button className="text-button">View full activity →</button>
    </section>
  );
}

function RecentTargets() {
  const targets = [
    ["user:123", "charge-card", "EXECUTING", "agent-1", "13:42:10", "00:00:08", "success"],
    ["user:123", "send-email", "EXECUTING", "agent-1", "13:42:11", "00:00:07", "success"],
    ["user:456", "generate-pdf", "WAITING", "—", "—", "—", "warning"],
    ["order:789", "charge-card", "RETRYING", "agent-2", "13:42:15", "00:00:05", "warning"],
    ["invoice:001", "send-email", "COMPLETED", "agent-1", "13:42:18", "00:00:01", "neutral"],
  ];

  return (
    <section className="recent-targets odyssey-panel">
      <PanelHeader title="RECENT TARGETS" action="View all →" />

      <div className="targets-table">
        <div className="table-header">
          <span>KEY</span>
          <span>TARGET</span>
          <span>STATE</span>
          <span>OWNER</span>
          <span>STARTED AT</span>
          <span>DURATION</span>
        </div>

        {targets.map((target, index) => (
          <div className="table-row" key={index}>
            <span>{target[0]}</span>
            <span>{target[1]}</span>
            <Status
              label={target[2]}
              tone={target[6]}
            />
            <span>{target[3]}</span>
            <span>{target[4]}</span>
            <span>{target[5]}</span>
          </div>
        ))}
      </div>
    </section>
  );
}

function PanelHeader({
  title,
  action,
}: {
  title: string;
  action?: string;
}) {
  return (
    <div className="panel-header">
      <span>{title}</span>

      {action && (
        <button className="panel-action">
          {action}
        </button>
      )}
    </div>
  );
}

function Status({
  label,
  tone,
}: {
  label: string;
  tone: string;
}) {
  return (
    <span className={`status status-${tone}`}>
      <span className="status-dot-small" />
      {label}
    </span>
  );
}
type ExecutionFiltersProps = {
  search: string;
  status: string;
  target: string;
  keyName: string;
  onSearchChange: (value: string) => void;
  onStatusChange: (value: string) => void;
  onTargetChange: (value: string) => void;
  onKeyChange: (value: string) => void;
};

export default function ExecutionFilters({
  search,
  status,
  target,
  keyName,
  onSearchChange,
  onStatusChange,
  onTargetChange,
  onKeyChange,
}: ExecutionFiltersProps) {
  return (
    <div className="execution-filters">
      <input
        type="text"
        value={search}
        onChange={(event) => onSearchChange(event.target.value)}
        placeholder="Search executions..."
        className="execution-search"
      />

      <select
        value={status}
        onChange={(event) => onStatusChange(event.target.value)}
        className="execution-filter"
      >
        <option value="all">Status: All</option>
        <option value="queued">Queued</option>
        <option value="claimed">Claimed</option>
        <option value="executing">Executing</option>
        <option value="completed">Completed</option>
        <option value="failed">Failed</option>
        <option value="uncertain">Uncertain</option>
        <option value="reconciling">Reconciling</option>
      </select>

      <select
        value={target}
        onChange={(event) => onTargetChange(event.target.value)}
        className="execution-filter"
      >
        <option value="all">Target: All</option>
        <option value="payment">Payment</option>
        <option value="order">Order</option>
        <option value="notification">Notification</option>
      </select>

      <select
        value={keyName}
        onChange={(event) => onKeyChange(event.target.value)}
        className="execution-filter"
      >
        <option value="all">Key: All</option>
        <option value="customer_123">customer_123</option>
        <option value="order_789">order_789</option>
      </select>
    </div>
  );
}
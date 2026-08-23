import ExecutionRow, {
  type Execution,
} from "./ExecutionRow";

type ExecutionListProps = {
  executions: Execution[];
  onExecutionClick?: (execution: Execution) => void;
};

export default function ExecutionList({
  executions,
  onExecutionClick,
}: ExecutionListProps) {
  return (
    <section className="execution-list odyssey-panel">
      <div className="execution-list-header">
        <span>EXECUTION</span>
        <span>STATUS</span>
        <span>ATTEMPT</span>
        <span>STARTED</span>
        <span>DURATION</span>
      </div>

      <div className="execution-list-body">
        {executions.length === 0 ? (
          <div className="execution-empty">
            No executions found.
          </div>
        ) : (
          executions.map((execution) => (
            <ExecutionRow
              key={execution.id}
              execution={execution}
              onClick={onExecutionClick}
            />
          ))
        )}
      </div>
    </section>
  );
}
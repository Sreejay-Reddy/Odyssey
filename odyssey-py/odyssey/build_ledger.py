from .results import BuildLedgerResult

class BuildLedger:
    def __init__(self, get_conn, key, steps):
        self.get_conn = get_conn
        self.key = key
        self.steps = steps

    def run(self):
        
        conn = self.get_conn()
        ledger_rows = []
        delivery_rows = []

        try:
            for step in self.steps:

                if step.delegate:
                    delivery_rows.append((self.key, step.target, step.delegate))
                    ledger_rows.append((self.key, step.target, "delegated", step.kwargs))
                else:
                    ledger_rows.append((self.key, step.target, "local", step.kwargs))

            with conn.cursor() as cur:
                cur.executemany("""
                INSERT INTO odyssey_ledger(
                key,
                target,
                mode,
                input
                )
                VALUES(%s,%s,%s,%s)
                """, ledger_rows, )

                cur.executemany("""
                INSERT INTO odyssey_deliveries(
                key,
                target,
                emit_to
                )
                VALUES(%s, %s, %s)
                """, delivery_rows, )

            conn.commit()

            return BuildLedgerResult(
                key=self.key,
                targets=[
                    step.target
                    for step in self.steps
                ],
                delegated=[
                    step.target
                    for step in self.steps
                    if step.delegate
                ],
                local=[
                    step.target
                    for step in self.steps
                    if not step.delegate
                ]
            )

        except Exception as e:
            conn.rollback()
            raise RuntimeError("Failed to build ledger") from e

        finally:
            conn.close()
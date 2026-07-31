class Build_ledger:
    def __init__(self, get_conn, key, steps, delegates):
        self.get_conn = get_conn
        self.key = key
        self.steps = steps
        self.delegates = delegates or {}

    def run(self):
            
        if not self.steps:
            raise ValueError("build_ledger requires at least one step")

        if len(self.steps) != len(set(self.steps)):
            raise ValueError("steps must be unique")

        unknown = set(self.delegates) - set(self.steps)
        if unknown:
            raise ValueError(f"delegates references steps not present in `steps`: {sorted(unknown)}")
        
        conn = self.get_conn()
        ledger_rows = []
        delivery_rows = []

        try:
            for target in self.steps:

                if target in self.delegates:
                    destination = self.delegates.get(target)
                    delivery_rows.append((self.key, target, destination))
                    ledger_rows.append((self.key, target, "delegated"))
                else:
                    ledger_rows.append((self.key, target, "local"))

            with conn.cursor() as cur:
                cur.executemany("""
                INSERT INTO odyssey_ledger(
                key,
                target,
                mode
                )
                VALUES(%s,%s,%s)
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
            return {
                "steps": len(ledger_rows),
                "delegated": len(delivery_rows)
            }

        except Exception as e:
            conn.rollback()
            raise RuntimeError("Failed to build ledger") from e

        finally:
            conn.close()
from .schema import SCHEMA_SQL

def init_db(conn):
    with conn.cursor() as cur:
        cur.execute(SCHEMA_SQL)

    conn.commit()

async def async_init_db(conn):
    async with conn.cursor() as cur:
        await cur.execute(SCHEMA_SQL)

    await conn.commit()
import os
import threading
import socket

def get_owner_id():
    hostname = socket.gethostname()
    pid = os.getpid()
    thread_id = threading.get_ident()

    return f"{hostname}:{pid}-{thread_id}"

def row_to_dict(cursor, row):
    columns = [desc[0] for desc in cursor.description]
    return dict(zip(columns, row))
import asyncio
import socket
from multiprocessing import Process
from .utils import decode_message, encode_result
from .execute import Execute
import time
import struct

class Worker:
    def __init__(self, worker_id, socket, resultsocket, registry):
        self.worker_id= worker_id
        self.socket = socket
        self.resultsocket = resultsocket
        self.registry = registry

    async def recv_exactly(self, size):
        loop = asyncio.get_running_loop()
        data = bytearray()

        while len(data) < size:
            chunk = await loop.sock_recv(
                self.socket,
                size - len(data),
            )

            if not chunk:
                raise ConnectionError("agent disconnected")

            data.extend(chunk)

        return bytes(data)

    async def collect_results(self, result_queue):
        while True:
            result = await result_queue.get()

            try:
                await self.send_result(result)
            finally:
                result_queue.task_done()

    async def send_result(self, result):
        data = encode_result(result)

        packet = struct.pack(">I", len(data)) + data

        loop = asyncio.get_running_loop()

        await loop.sock_sendall(
            self.resultsocket,
            packet,
        )

    async def run_execution(self, execution, result_queue):
        try:
            result = await Execute(
                execution.key,
                execution.target_id,
                execution.input,
                self.registry,
            ).run()

            await result_queue.put(result)
            
        except asyncio.CancelledError:
            raise

        except Exception as e:
            # construct FAILED ResultExecution here
            pass

    async def run(self):
        self.socket.setblocking(False)
        self.resultsocket.setblocking(False)

        try:
            tasks = set()
            result_queue = asyncio.Queue()

            result_collector = asyncio.create_task(
                self.collect_results(result_queue)
            )

            while True:
                header = await self.recv_exactly(4)
                size = int.from_bytes(header, "big")

                data = await self.recv_exactly(size)
                message = decode_message(data)

                for execution in message.executions:
                    task = asyncio.create_task(
                        self.run_execution(execution, result_queue)
                    )
                    tasks.add(task)
                    task.add_done_callback(tasks.discard)

        except asyncio.CancelledError:
            raise

        except ConnectionError:
            pass

        finally:
            result_collector.cancel()

            for task in tasks:
                task.cancel()

            await asyncio.gather(
                result_collector,
                *tasks,
                return_exceptions=True,
            )

            self.socket.close()
            self.resultsocket.close()

    def worker_process(self):
        asyncio.run(self.run())

    def stop(self):
        self.socket.close()


class WorkerPool:
    def __init__(self, registry, workers = 4):
        self.workers = workers
        self.registry = registry
        self.processes = []
        self.worker_instances = []

    def connect_worker(self, worker_id):
        path = f"/tmp/odyssey/{worker_id}.sock"

        while True:
            sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

            try:
                sock.connect(path)
                return sock
            except ConnectionRefusedError:
                sock.close()
                time.sleep(1)

    def connect_result(self, result_id):
        path = f"/tmp/odyssey/{result_id}.sock"

        while True:
            sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)

            try:
                sock.connect(path)
                return sock
            except ConnectionRefusedError:
                sock.close()
                time.sleep(1)


    def create_workers(self):
        for worker in range(self.workers):
            worker_id = f"worker-{worker}"
            result_id = f"result-{worker}"

            sock = self.connect_worker(worker_id)
            result_sock = self.connect_result(result_id)

            worker_instance = Worker(
                worker_id=worker_id, 
                resultsocket=result_sock,
                socket=sock,
                registry=self.registry
            )

            process = Process(
                target=worker_instance.worker_process,
            )
            process.start()

            self.worker_instances.append(worker_instance)
            self.processes.append(process)

            sock.close()
            result_sock.close()

    def wait(self):
        for process in self.processes:
            process.join()

    def stop(self):
        for worker in self.worker_instances:
            worker.stop()

        for process in self.processes:
            if process.is_alive():
                process.terminate()

        for process in self.processes:
            process.join()

        self.worker_instances.clear()
        self.processes.clear()
import asyncio
from .execute import Execute

class Job:
    def __init__(self, key, target, future):
        self.key = key
        self.target = target
        self.future = future


class Worker:
    def __init__(self, queue, get_conn, registry):
        self.queue = queue
        self.get_conn = get_conn
        self.registry = registry

    async def run(self):
        while True:
            job = await self.queue.get()

            try:
                registered = self.registry.get(job.target)

                execution = Execute(
                    get_conn=self.get_conn,
                    key=job.key,
                    target=job.target,
                    fn=registered.fn,
                    ttl_ms=registered.ttl_ms,
                )

                result = await execution.run()

                job.future.set_result(result)

            except Exception as e:
                job.future.set_exception(e)

            finally:
                self.queue.task_done()


class WorkerPool:
    def __init__(self, worker_count, get_conn, registry):
        self.get_conn = get_conn
        self.registry = registry

        self.queue = asyncio.Queue()
        self.workers = []
        self.tasks = []

        for _ in range(worker_count):
            worker = Worker(
                self.queue,
                self.get_conn,
                self.registry,
            )

            self.workers.append(worker)

    def start(self):
        for worker in self.workers:
            task = asyncio.create_task(worker.run())
            self.tasks.append(task)

    async def stop(self):
        for task in self.tasks:
            task.cancel()

        await asyncio.gather(
            *self.tasks,
            return_exceptions=True,
        )
        
        self.tasks.clear()

    async def publish(self, key, target):
        loop = asyncio.get_running_loop()
        future = loop.create_future()

        job = Job(
            key=key,
            target=target,
            future=future,
        )

        await self.queue.put(job)

        return await future
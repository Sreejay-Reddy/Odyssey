# Response structure of acquire
class AcquireResult:
    def __init__(self, acquired, target, owner_id=None, expires_at=None, fencing_token=None, status=None, journey_alive=None):
        self.acquired = acquired
        self.owner_id = owner_id
        self.target = target
        self.expires_at = expires_at
        self.fencing_token = fencing_token
        self.status = status
        self.journey_alive = journey_alive

class OperationResult:
    def __init__(self, success: bool, status=None):
        self.success = success
        self.status = status

class InspectResult:
    def __init__(
        self, key, owner_id, fencing_token, status, journey_alive, 
        expires_at, updated_at, execution_result=None
    ):
        self.key = key
        self.owner_id = owner_id
        self.fencing_token = fencing_token
        self.status = status
        self.journey_alive = journey_alive
        self.expires_at = expires_at
        self.updated_at = updated_at
        self.execution_result = execution_result

class BuildLedgerResult:
    def __init__(
        self,
        key,
        targets,
        delegated,
        local,
    ):
        self.key = key
        self.targets = targets
        self.delegated = delegated
        self.local = local

    @property
    def step_count(self):
        return len(self.targets)
        

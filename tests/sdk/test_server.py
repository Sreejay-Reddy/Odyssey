from starlette.testclient import TestClient

from odyssey.server import OdysseyServer


def test_execute_endpoint_uses_ledger_input(
    clean_database,
    odyssey,
    ledger,
    pool,
):
    server = OdysseyServer(
        registry=odyssey._register,
        pool=pool,
    )

    client = TestClient(server.app)

    response = client.post(
        "/execute",
        json={
            "key": ledger["key"],
            "target": ledger["target"],
        },
    )

    assert response.status_code == 200

    data = response.json()

    assert data["status"] == "completed"
    assert data["key"] == ledger["key"]
    assert data["target"] == ledger["target"]
    assert data["result"] == "Hello, Summer!"


def test_execute_does_not_accept_agent_input_as_source_of_truth(
    clean_database,
    odyssey,
    ledger,
    pool,
):
    server = OdysseyServer(
        registry=odyssey._register,
        pool=pool,
    )

    client = TestClient(server.app)

    response = client.post(
        "/execute",
        json={
            "key": ledger["key"],
            "target": ledger["target"],
            "input": {
                "name": "Attacker",
            },
        },
    )

    assert response.status_code == 200

    data = response.json()

    # The ledger contains {"name": "Summer"}.
    # The request's input must not override it.
    assert data["result"] == "Hello, Summer!"


def test_invalid_json(
    odyssey,
    pool,
):
    server = OdysseyServer(
        registry=odyssey._register,
        pool=pool,
    )

    client = TestClient(server.app)

    response = client.post(
        "/execute",
        content=b"not-json",
        headers={
            "Content-Type": "application/json",
        },
    )

    assert response.status_code == 400
    assert response.json()["error"] == "Invalid JSON payload."


def test_missing_key(
    odyssey,
    pool,
):
    server = OdysseyServer(
        registry=odyssey._register,
        pool=pool,
    )

    client = TestClient(server.app)

    response = client.post(
        "/execute",
        json={
            "target": "hello",
        },
    )

    assert response.status_code == 400
    assert response.json()["error"] == "Missing key."


def test_missing_target(
    odyssey,
    pool,
):
    server = OdysseyServer(
        registry=odyssey._register,
        pool=pool,
    )

    client = TestClient(server.app)

    response = client.post(
        "/execute",
        json={
            "key": "test",
        },
    )

    assert response.status_code == 400
    assert response.json()["error"] == "Missing target."


def test_unknown_target(
    odyssey,
    pool,
):
    server = OdysseyServer(
        registry=odyssey._register,
        pool=pool,
    )

    client = TestClient(server.app)

    response = client.post(
        "/execute",
        json={
            "key": "test",
            "target": "missing",
        },
    )

    assert response.status_code == 404
    assert "not registered" in response.json()["error"]


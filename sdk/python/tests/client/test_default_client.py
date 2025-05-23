import pytest

import dagger
from dagger import dag
from dagger.client._connection import SharedConnection

pytestmark = [
    pytest.mark.anyio,
    pytest.mark.slow,
]


def test_singleton():
    assert SharedConnection() is SharedConnection()


async def test_context_manager_provision(alpine_image: str):
    conn = SharedConnection()
    assert not conn.is_connected(), "Connection should not be established yet."

    async with dagger.connection():
        assert conn.is_connected(), "Connection should be established."
        out = (
            await dag.container()
            .from_(alpine_image)
            .with_exec(["echo", "-n", "hello"])
            .stdout()
        )

    assert out == "hello"
    assert not conn.is_connected(), "Connection should be closed."


class TestConnectionManagement:
    @pytest.fixture(scope="class", autouse=True)
    async def _setup(self):
        from dagger.provisioning._engine import provision_engine

        # This allows running these tests from the host by auto provisioning.
        async with provision_engine(dagger.Config(retry=None)) as engine:
            # Just setup connection, don't connect yet.
            # We want to test connect and disconnect.
            conn = engine.get_shared_client_connection()

            conn_params = conn._params
            engine_params = engine.connect_params

            assert conn_params is not None, "Connection params should be set."
            assert engine_params is not None, "Engine params should be set."
            assert conn_params.url == engine_params.url

            yield

    @pytest.fixture(autouse=True)
    async def _assert_connection_status(self):
        conn = SharedConnection()
        # Each test should start and end with no established client session.
        assert not conn.is_connected(), "Connection should not be established yet."
        yield
        assert not conn.is_connected(), "Connection should be closed."

    async def test_connect_with_context_manager(self, alpine_image: str):
        async with await dagger.connect() as conn:
            assert conn.is_connected(), "Connection should be established."
            out = await (
                dag.container()
                .from_(alpine_image)
                .with_exec(["echo", "-n", "hello"])
                .stdout()
            )
            assert out == "hello"

    async def test_connect_and_close(self, alpine_image: str):
        conn = await dagger.connect()
        assert conn.is_connected(), "Connection should be established."
        out = (
            await dag.container()
            .from_(alpine_image)
            .with_exec(["echo", "-n", "hello"])
            .stdout()
        )
        assert out == "hello"
        await conn.close()

    async def test_connect_and_global_close(self, alpine_image: str):
        await dagger.connect()
        out = (
            await dag.container()
            .from_(alpine_image)
            .with_exec(["echo", "-n", "hello"])
            .stdout()
        )
        assert out == "hello"
        await dagger.close()

    async def test_lazy_connect_and_global_close(self, alpine_image: str):
        out = (
            await dag.container()
            .from_(alpine_image)
            .with_exec(["echo", "-n", "hello"])
            .stdout()
        )
        assert out == "hello"
        await dagger.close()

import contextlib
import logging
import os
import typing

from typing_extensions import Self

import dagger
from dagger.client._session import (
    BaseConnection,
    ConnectConfig,
    ConnectParams,
    SharedConnection,
    SingleConnection,
)

from ._config import Config
from ._download import Downloader
from ._exceptions import ProvisionError
from ._progress import Progress
from ._session import start_cli_session

logger = logging.getLogger(__name__)

if typing.TYPE_CHECKING:
    from dagger import Client


@contextlib.asynccontextmanager
async def provision_engine(cfg: Config):
    """Provision a new engine session."""
    async with contextlib.AsyncExitStack() as stack:
        logger.debug("Provisioning engine")
        yield await Engine(cfg, stack).provision()
        logger.debug("Closing engine provisioning")


class Engine:
    """Start engine session, provisioning if needed."""

    def __init__(self, cfg: Config, stack: contextlib.AsyncExitStack) -> None:
        super().__init__()
        self.cfg = cfg
        self.stack = stack
        self.progress = Progress(cfg.console)
        self.connect_params = None
        self.connect_config = None
        self.has_provisioned = False

    async def provision(self) -> Self:
        connect_params = ConnectParams.from_env()

        if connect_params and self.cfg.workdir:
            msg = (
                "Cannot configure workdir for existing session "
                "(please use --workdir or host.directory "
                "with absolute paths instead)."
            )
            raise ProvisionError(msg)

        if not connect_params:
            self.has_provisioned = True
            # Only start progress if we are provisioning, not on active sessions
            # like `dagger run`.
            await self.progress.start("Provisioning engine")
            cli_bin = await self.get_cli()

            await self.progress.update("Creating new Engine session")
            connect_params = await self.stack.enter_async_context(
                start_cli_session(self.cfg, cli_bin)
            )

        self.connect_params = connect_params
        self.connect_config = ConnectConfig(
            timeout=self.cfg.timeout,
            retry=self.cfg.retry,
        )

        return self

    async def get_cli(self) -> str:
        """Get path to CLI."""
        if cli_bin := os.getenv("_EXPERIMENTAL_DAGGER_CLI_BIN"):
            return cli_bin

        # Get from cache or download.
        return await Downloader(progress=self.progress)

    async def setup_client(self, conn: BaseConnection) -> "Client":
        """Setup client instance from connection."""
        await self.progress.update("Establishing connection to the API server")
        conn = await self.stack.enter_async_context(conn)

        client = dagger.Client.from_connection(conn)
        self.stack.push_async_callback(self.progress.update, "Disconnecting")

        return await self.verify(client)

    def get_shared_client_connection(self) -> SharedConnection:
        """Global client connection to the GraphQL server."""
        assert self.connect_params
        assert self.connect_config
        return (
            SharedConnection()
            .with_params(self.connect_params)
            .with_config(self.connect_config)
        )

    def get_client_connection(self) -> SingleConnection:
        """Isolated client connection to the GraphQL server."""
        assert self.connect_params
        assert self.connect_config
        return SingleConnection(
            self.connect_params,
            self.connect_config,
        )

    async def verify(self, client: "Client") -> "Client":
        """Check if the Dagger CLI version is compatible with the engine."""
        await self.progress.update("Checking version compatibility")
        try:
            await client.version()
        except dagger.QueryError as e:
            logger.warning("Failed to check Dagger engine version compatibility: %s", e)

        # If log_output is set, we don't need to show any more progress.
        if self.cfg.log_output:
            await self.progress.stop()
        else:
            await self.progress.update("Running pipelines")

        return client

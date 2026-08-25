#!/usr/bin/env python3
"""Toggle the PLC ftaReady input with the R key.

Usage:
    python fta_ready_toggle.py [--host HOST] [--port PORT]

Controls:
    r  Toggle ftaReady (PLC input channel 19)
    q  Quit

Install ``readchar`` for immediate key detection. Without it, enter ``r`` or
``q`` followed by Enter.
"""

import argparse
import asyncio
import json
import logging
import signal
import sys
import threading
from typing import Any

import websockets


FTA_READY_CHANNEL = 19

logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%H:%M:%S",
)
log = logging.getLogger("fta_ready_toggle")


def encode_ws_message(message_type: str, data: Any) -> str:
    return json.dumps({"type": message_type, "data": data})


class FTAReadyToggle:
    def __init__(self, host: str, port: int) -> None:
        self.ws_url = f"ws://{host}:{port}/api/plc/websocket"
        self._ws = None
        self._stopping = False
        self.state = False
        self._state_known = False

    async def toggle(self) -> None:
        if self._ws is None:
            log.warning("Not connected; ftaReady was not changed")
            return
        if not self._state_known:
            log.warning("Waiting for the initial PLC input state")
            return

        new_state = not self.state
        message = encode_ws_message(
            "setInput",
            {"channel": FTA_READY_CHANNEL, "state": new_state},
        )

        try:
            await self._ws.send(message)
        except websockets.exceptions.WebSocketException as exc:
            log.warning("Could not set ftaReady: %s", exc)
            return

        self.state = new_state
        log.info("ftaReady (input %d) -> %s", FTA_READY_CHANNEL, self.state)

    async def connect(self) -> None:
        log.info("Connecting to %s", self.ws_url)
        async with websockets.connect(
            self.ws_url,
            ping_interval=20,
            ping_timeout=30,
        ) as ws:
            self._ws = ws
            log.info("Connected. Press 'r' to toggle ftaReady; press 'q' to quit.")

            try:
                async for raw in ws:
                    try:
                        message = json.loads(raw)
                    except (json.JSONDecodeError, TypeError):
                        log.warning("Ignoring invalid websocket message: %r", raw)
                        continue

                    message_type = message.get("type", "") if isinstance(message, dict) else ""
                    data = message.get("data") if isinstance(message, dict) else None
                    if message_type == "plcIoChange" and isinstance(data, dict):
                        inputs = data.get("Inputs", data.get("inputs"))
                        if isinstance(inputs, list) and len(inputs) > FTA_READY_CHANNEL:
                            actual_state = bool(inputs[FTA_READY_CHANNEL])
                            if not self._state_known or actual_state != self.state:
                                self.state = actual_state
                                log.info(
                                    "Current ftaReady (input %d): %s",
                                    FTA_READY_CHANNEL,
                                    self.state,
                                )
                            self._state_known = True
                    elif message_type == "plcInputSetSuccess":
                        log.debug("Server acknowledged input change: %s", data)
                    elif message_type == "error":
                        log.warning("Server error: %s", data)
            finally:
                self._ws = None
                self._state_known = False
                log.info("Disconnected")

    async def run_forever(self, reconnect_delay: float = 3.0) -> None:
        while not self._stopping:
            try:
                await self.connect()
            except (OSError, websockets.exceptions.WebSocketException) as exc:
                if self._stopping:
                    break
                log.warning(
                    "Connection lost: %s. Retrying in %.0f seconds",
                    exc,
                    reconnect_delay,
                )
                await asyncio.sleep(reconnect_delay)

    async def stop(self) -> None:
        self._stopping = True
        if self._ws is not None:
            await self._ws.close()


def keyboard_listener(
    client: FTAReadyToggle,
    loop: asyncio.AbstractEventLoop,
) -> None:
    try:
        import readchar

        read_key = readchar.readkey
        print("Keyboard ready. Press 'r' to toggle ftaReady; press 'q' to quit.")
    except ImportError:
        read_key = lambda: sys.stdin.readline().strip()
        print(
            "Keyboard ready (press Enter after each key). "
            "Install readchar for immediate key detection."
        )

    while True:
        try:
            key = read_key()
        except (EOFError, KeyboardInterrupt):
            key = "q"

        key = key.lower()
        if key == "r":
            asyncio.run_coroutine_threadsafe(client.toggle(), loop)
        elif key == "q" or key == "":
            asyncio.run_coroutine_threadsafe(client.stop(), loop)
            return
        else:
            print("Unknown key. Use 'r' to toggle or 'q' to quit.")


async def main(host: str, port: int) -> None:
    client = FTAReadyToggle(host, port)
    loop = asyncio.get_running_loop()

    def handle_signal() -> None:
        asyncio.create_task(client.stop())

    for signal_name in (signal.SIGINT, signal.SIGTERM):
        try:
            loop.add_signal_handler(signal_name, handle_signal)
        except NotImplementedError:
            signal.signal(signal_name, lambda *_: loop.call_soon_threadsafe(handle_signal))

    threading.Thread(
        target=keyboard_listener,
        args=(client, loop),
        daemon=True,
        name="keyboard-listener",
    ).start()

    await client.run_forever()


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="Toggle the PLC ftaReady input")
    parser.add_argument("--host", default="localhost", help="Arena server host")
    parser.add_argument("--port", default=8080, type=int, help="Arena server port")
    args = parser.parse_args()

    try:
        asyncio.run(main(args.host, args.port))
    except KeyboardInterrupt:
        pass

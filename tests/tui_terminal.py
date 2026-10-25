"""Bounded native PTY driver shared by TUI integration and visual checks."""
import os
import pathlib
import pty
import select
import struct
import subprocess
import threading
import time
import fcntl
import termios
import signal
import re
import codecs

ANSI_CSI = re.compile(rb"\x1b\[[0-?]*[ -/]*[@-~]")

BINARY = str(pathlib.Path(os.environ['NIDO_TEST_BINARY']).resolve())
TIMEOUT = 8

class TUI:
    def __init__(self, environment, cwd, size=(150,45), observer=None):
        self.observer = observer
        self.decoder = codecs.getincrementaldecoder("utf-8")("replace")
        self.master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', size[1], size[0], 0, 0))
        self.process = subprocess.Popen([BINARY, 'gui'], env=environment, cwd=cwd,
                                        stdin=slave, stdout=slave, stderr=slave,
                                        start_new_session=True)
        os.close(slave)
        self.output = bytearray()
        self.stop = threading.Event()
        self.thread = threading.Thread(target=self.read, daemon=True)
        self.thread.start()

    def read(self):
        while not self.stop.is_set():
            if select.select([self.master], [], [], .1)[0]:
                try:
                    data = os.read(self.master, 65536)
                except OSError:
                    return
                if not data:
                    return
                self.output.extend(data)
                if self.observer is not None:
                    self.observer.feed(self.decoder.decode(data))
                assert len(self.output) < 2 * 1024 * 1024, 'unbounded PTY trace'

    def resize(self, width, height):
        if self.observer is not None:
            self.observer.screen.resize(lines=height, columns=width)
        fcntl.ioctl(self.master, termios.TIOCSWINSZ,
                    struct.pack('HHHH', height, width, 0, 0))
        os.kill(self.process.pid, signal.SIGWINCH)

    def key(self, key):
        os.write(self.master, key)

    def text(self, value, after=0):
        deadline = time.monotonic() + TIMEOUT
        while time.monotonic() < deadline:
            if value.encode() in ANSI_CSI.sub(b"", bytes(self.output[after:])):
                return
            assert self.process.poll() is None, self.output.decode(errors='replace')[-2000:]
            time.sleep(.025)
        raise AssertionError(f'TUI did not render {value!r}: ' + self.output.decode(errors='replace')[-4000:])

    def close(self):
        self.key(b'\x03')
        try:
            self.process.wait(timeout=TIMEOUT)
        finally:
            if self.process.poll() is None:
                self.process.kill()
                self.process.wait()
            self.stop.set()
            self.thread.join(timeout=TIMEOUT)
            os.close(self.master)
        assert self.process.returncode == 0, self.output.decode(errors='replace')[-2000:]

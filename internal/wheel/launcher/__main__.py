"""python -m pftw ARGS: run pftw even when pip's Scripts folder is not on PATH."""

import subprocess
import sys

from pftw import executable


def main():
    try:
        exe = executable()
    except FileNotFoundError as error:
        sys.stderr.write("%s\n" % error)
        return 1
    process = subprocess.Popen([exe] + sys.argv[1:])
    while True:
        try:
            return process.wait()
        except KeyboardInterrupt:
            # pftw receives the same Ctrl+C and finishes or rolls back its own work; wait for it.
            continue


if __name__ == "__main__":
    sys.exit(main())

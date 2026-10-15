"""pftw: install official Python on Windows without the Store, MSI or Python Install Manager.

This package only carries the pftw executable. pftw itself is written in Go, not Python.
"""

import os
import sysconfig


def executable():
    """Return the path of the pftw.exe installed with this package."""
    folders = [sysconfig.get_path("scripts")]
    try:
        user = sysconfig.get_preferred_scheme("user")
    except AttributeError:  # Python < 3.10
        user = os.name + "_user"
    try:
        folders.append(sysconfig.get_path("scripts", user))
    except KeyError:
        pass
    # Unusual layouts: bin inside a pip --target folder, Scripts beside site-packages (user)
    # or beside Lib (prefix).
    package = os.path.dirname(os.path.abspath(__file__))
    site = os.path.dirname(package)
    folders.append(os.path.join(site, "bin"))
    folders.append(os.path.join(os.path.dirname(site), "Scripts"))
    folders.append(os.path.join(os.path.dirname(os.path.dirname(site)), "Scripts"))
    for folder in folders:
        candidate = os.path.join(folder, "pftw.exe")
        if os.path.isfile(candidate):
            return candidate
    raise FileNotFoundError("pftw.exe was not found; reinstall with: python -m pip install --force-reinstall pftw")
